package services

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	appconfig "github.com/365Soft-Bolivia/inmobiliaria-backend/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// StorageService - Servicio para almacenamiento de archivos en Cloudflare R2
type StorageService struct {
	client *s3.Client
	cfg    appconfig.R2Config
}

// NewStorageService - Crea un nuevo servicio de almacenamiento
func NewStorageService(cfg appconfig.R2Config) (*StorageService, error) {
	// Verificar si la configuración de R2 es válida (no placeholders ni vacía)
	isPlaceholder := cfg.AccountID == "tu_account_id" ||
		cfg.AccessKeyID == "tu_access_key" ||
		cfg.SecretKey == "tu_secret_key" ||
		cfg.BucketName == "tu_bucket_name"

	isEmpty := cfg.AccountID == "" ||
		cfg.AccessKeyID == "" ||
		cfg.SecretKey == "" ||
		cfg.BucketName == ""

	if isEmpty || isPlaceholder {
		log.Println("⚠️  Configuración de R2 incompleta o con placeholders, usando almacenamiento local")
		return &StorageService{cfg: cfg}, nil
	}

	// Configurar el cliente de AWS SDK para R2
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:           fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID),
			SigningRegion: "auto",
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithEndpointResolverWithOptions(r2Resolver),
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretKey,
			}, nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("error configurando cliente AWS: %w", err)
	}

	client := s3.NewFromConfig(awsCfg)

	log.Println("✅ Servicio de almacenamiento R2 inicializado")
	return &StorageService{
		client: client,
		cfg:    cfg,
	}, nil
}

// UploadFile - Sube un archivo a R2 o almacenamiento local
func (s *StorageService) UploadFile(file *multipart.FileHeader, userID uint, documentType string) (string, int64, error) {
	// Abrir el archivo
	src, err := file.Open()
	if err != nil {
		return "", 0, fmt.Errorf("error abriendo archivo: %w", err)
	}
	defer src.Close()

	// Generar nombre único para el archivo
	timestamp := time.Now().Format("20060102_150405")
	fileExtension := filepath.Ext(file.Filename)
	uniqueFileName := fmt.Sprintf("%d_%s_%s%s", userID, documentType, timestamp, fileExtension)

	// Generar ruta del archivo
	filePath := fmt.Sprintf("documents/%s/%s", documentType, uniqueFileName)

	// Leer contenido del archivo
	fileData, err := io.ReadAll(src)
	if err != nil {
		return "", 0, fmt.Errorf("error leyendo archivo: %w", err)
	}

	// Determinar si usar R2 o almacenamiento local
	if s.client != nil {
		// Subir a R2
		fileURL, err := s.uploadToR2(filePath, fileData, file.Header.Get("Content-Type"))
		if err != nil {
			return "", 0, err
		}
		return fileURL, file.Size, nil
	}

	// Usar almacenamiento local
	fileURL, err := s.uploadToLocal(filePath, fileData, uniqueFileName)
	if err != nil {
		return "", 0, err
	}
	return fileURL, file.Size, nil
}

// uploadToR2 - Sube un archivo a Cloudflare R2
func (s *StorageService) uploadToR2(filePath string, fileData []byte, contentType string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Subir archivo a R2
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.BucketName),
		Key:         aws.String(filePath),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACL(*aws.String("public-read")), // Archivos públicos
	})
	if err != nil {
		return "", fmt.Errorf("error subiendo archivo a R2: %w", err)
	}

	// Generar URL pública del archivo
	publicURL := fmt.Sprintf("https://pub-%s.r2.dev/%s", s.cfg.AccountID, filePath)

	return publicURL, nil
}

// uploadToLocal - Sube un archivo al almacenamiento local
func (s *StorageService) uploadToLocal(filePath string, fileData []byte, fileName string) (string, error) {
	// Crear directorio si no existe
	fullPath := filepath.Join(".", "uploads", "documents")
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return "", fmt.Errorf("error creando directorio: %w", err)
	}

	// Guardar archivo localmente
	filePath = filepath.Join(fullPath, fileName)
	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return "", fmt.Errorf("error guardando archivo localmente: %w", err)
	}

	// Retornar URL relativa
	fileURL := fmt.Sprintf("/uploads/documents/%s", fileName)

	return fileURL, nil
}

// DeleteFile - Elimina un archivo de R2 o almacenamiento local
func (s *StorageService) DeleteFile(fileURL string) error {
	// Extraer el nombre del archivo de la URL
	var fileName string
	if strings.Contains(fileURL, "r2.dev") {
		// URL de R2: https://pub-{account_id}.r2.dev/documents/{type}/{filename}
		parts := strings.Split(fileURL, "/")
		if len(parts) >= 2 {
			fileName = strings.Join(parts[len(parts)-2:], "/")
		}
	} else {
		// URL local: /uploads/documents/{filename}
		parts := strings.Split(fileURL, "/")
		if len(parts) >= 1 {
			fileName = parts[len(parts)-1]
		}
	}

	if fileName == "" {
		return fmt.Errorf("no se pudo extraer el nombre del archivo de la URL")
	}

	// Determinar si usar R2 o almacenamiento local
	if s.client != nil && strings.Contains(fileURL, "r2.dev") {
		// Eliminar de R2
		return s.deleteFromR2(fileName)
	}

	// Eliminar de almacenamiento local
	return s.deleteFromLocal(fileName)
}

// deleteFromR2 - Elimina un archivo de Cloudflare R2
func (s *StorageService) deleteFromR2(filePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.BucketName),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return fmt.Errorf("error eliminando archivo de R2: %w", err)
	}

	log.Printf("Archivo eliminado de R2: %s", filePath)
	return nil
}

// deleteFromLocal - Elimina un archivo del almacenamiento local
func (s *StorageService) deleteFromLocal(fileName string) error {
	filePath := filepath.Join(".", "uploads", "documents", fileName)

	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Archivo no encontrado en almacenamiento local: %s", filePath)
			return nil
		}
		return fmt.Errorf("error eliminando archivo local: %w", err)
	}

	log.Printf("Archivo eliminado de almacenamiento local: %s", filePath)
	return nil
}

// GetFileURL - Obtiene la URL pública de un archivo
func (s *StorageService) GetFileURL(filePath string) string {
	if s.client != nil {
		return fmt.Sprintf("https://pub-%s.r2.dev/%s", s.cfg.AccountID, filePath)
	}
	return fmt.Sprintf("/uploads/documents/%s", filepath.Base(filePath))
}

// IsUsingR2 - Retorna true si está usando R2
func (s *StorageService) IsUsingR2() bool {
	return s.client != nil
}
