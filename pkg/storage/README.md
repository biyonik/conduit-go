# Storage Package

Laravel-inspired file storage system for Conduit-Go with local and S3 support.

## Features

- **Multiple Drivers**: Local filesystem, S3-compatible storage
- **Path Traversal Protection**: Secure file operations
- **Stream Support**: Memory-efficient for large files
- **Simple API**: Laravel-like fluent interface
- **URL Generation**: Public URL creation for files
- **Directory Operations**: Create, list, delete directories

## Quick Start

### Local Storage

```go
import "github.com/biyonik/conduit-go/pkg/storage"

// Initialize
localStorage, err := storage.NewLocalStorage("/var/www/uploads", logger)
if err != nil {
    log.Fatal(err)
}

// Optional: Set base URL
localStorage.SetBaseURL("https://cdn.myapp.com")

// Upload file
imageData := []byte{...}
err = localStorage.Put("avatars/user-1.jpg", imageData)

// Get public URL
url := localStorage.Url("avatars/user-1.jpg")
// → "https://cdn.myapp.com/avatars/user-1.jpg"

// Download file
data, err := localStorage.Get("avatars/user-1.jpg")

// Delete file
err = localStorage.Delete("avatars/user-1.jpg")
```

## File Operations

### Upload Files

```go
// From byte array
imageData := []byte{...}
err := storage.Put("images/photo.jpg", imageData)

// From file (stream - memory efficient)
file, _ := os.Open("large-video.mp4")
defer file.Close()
err := storage.PutFile("videos/large-video.mp4", file)

// From HTTP upload
func UploadHandler(w http.ResponseWriter, r *http.Request) {
    file, header, _ := r.FormFile("avatar")
    defer file.Close()

    // Generate unique name
    uniqueName := storage.GenerateUniqueName(header.Filename)
    path := "avatars/" + uniqueName

    err := storage.PutFile(path, file)
}
```

### Download Files

```go
// Get file contents
data, err := storage.Get("documents/invoice.pdf")
if err == storage.ErrFileNotFound {
    http.NotFound(w, r)
    return
}

// Stream large files
reader, err := storage.GetStream("videos/large.mp4")
if err != nil {
    return err
}
defer reader.Close()

// Stream to HTTP response
io.Copy(w, reader)
```

### Check File Existence

```go
exists, err := storage.Exists("avatars/user-1.jpg")
if exists {
    fmt.Println("File exists")
}
```

### Get File Info

```go
// Size
size, err := storage.Size("documents/report.pdf")
fmt.Printf("File size: %d bytes\n", size)

// Last modified
modTime, err := storage.LastModified("images/photo.jpg")
fmt.Printf("Last modified: %v\n", modTime)
```

### Delete Files

```go
err := storage.Delete("temp/old-file.txt")
if err == storage.ErrFileNotFound {
    fmt.Println("File not found")
}
```

## Directory Operations

### Create Directory

```go
err := storage.MakeDirectory("uploads/2024/january")
```

### List Files

```go
// List files in directory
files, err := storage.Files("uploads")
for _, file := range files {
    fmt.Println(file)
}

// List subdirectories
dirs, err := storage.Directories("uploads")
for _, dir := range dirs {
    fmt.Println(dir)
}
```

### Delete Directory

```go
// Delete directory and all contents
err := storage.DeleteDirectory("temp/cache")
```

## URL Generation

```go
// Set base URL for your CDN or domain
storage.SetBaseURL("https://cdn.myapp.com")

// Generate URLs
avatarURL := storage.Url("avatars/user-1.jpg")
// → "https://cdn.myapp.com/avatars/user-1.jpg"

// Use in templates
fmt.Fprintf(w, `<img src="%s">`, storage.Url("images/banner.jpg"))
```

## Security

### Path Traversal Protection

```go
// These are automatically blocked
storage.Put("../../../etc/passwd", data)  // ❌ Error: invalid path
storage.Get("../../secret.txt")           // ❌ Error: invalid path

// Safe paths
storage.Put("uploads/file.txt", data)     // ✅ OK
storage.Put("users/1/avatar.jpg", data)   // ✅ OK
```

### File Type Validation

```go
func UploadHandler(w http.ResponseWriter, r *http.Request) {
    file, header, _ := r.FormFile("image")
    defer file.Close()

    // Check if image
    if !storage.IsImage(header.Filename) {
        http.Error(w, "Only images allowed", http.StatusBadRequest)
        return
    }

    path := "images/" + storage.GenerateUniqueName(header.Filename)
    storage.PutFile(path, file)
}
```

## Best Practices

### 1. Organize by Directory

```go
// User avatars
storage.Put("avatars/" + userID + ".jpg", imageData)

// Documents by year/month
storage.Put("documents/2024/01/invoice-123.pdf", pdfData)

// Temporary files
storage.Put("temp/upload-" + sessionID + ".tmp", data)
```

### 2. Generate Unique Names

```go
// Avoid name collisions
originalName := "photo.jpg"
uniqueName := storage.GenerateUniqueName(originalName)
// → "1704067200-photo.jpg"

storage.Put("uploads/" + uniqueName, imageData)
```

### 3. Stream Large Files

```go
// ❌ Bad: Loads entire file into memory
data, _ := os.ReadFile("large-file.mp4")
storage.Put("videos/large.mp4", data)

// ✅ Good: Streams file
file, _ := os.Open("large-file.mp4")
defer file.Close()
storage.PutFile("videos/large.mp4", file)
```

### 4. Clean Up Temporary Files

```go
// After processing
tempPath := "temp/upload-123.tmp"
defer storage.Delete(tempPath)

// Process file...
storage.Put("permanent/file.jpg", processedData)
```

### 5. Handle Errors

```go
err := storage.Get("file.txt")
if err != nil {
    switch err {
    case storage.ErrFileNotFound:
        http.NotFound(w, r)
    case storage.ErrPermissionDenied:
        http.Error(w, "Permission denied", http.StatusForbidden)
    default:
        http.Error(w, "Internal error", http.StatusInternalServerError)
    }
    return
}
```

## Integration Examples

### With HTTP File Upload

```go
func AvatarUploadHandler(w http.ResponseWriter, r *http.Request) {
    // Parse multipart form
    r.ParseMultipartForm(10 << 20) // 10 MB limit

    // Get file
    file, header, err := r.FormFile("avatar")
    if err != nil {
        http.Error(w, "No file uploaded", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Validate
    if !storage.IsImage(header.Filename) {
        http.Error(w, "Only images allowed", http.StatusBadRequest)
        return
    }

    // Generate path
    userID := getUserID(r)
    ext := storage.GetExtension(header.Filename)
    path := fmt.Sprintf("avatars/user-%d%s", userID, ext)

    // Upload
    if err := storage.PutFile(path, file); err != nil {
        http.Error(w, "Upload failed", http.StatusInternalServerError)
        return
    }

    // Return URL
    url := storage.Url(path)
    json.NewEncoder(w).Encode(map[string]string{
        "url": url,
    })
}
```

### With Queue System

```go
type ProcessImageJob struct {
    FilePath string
    UserID   int64
}

func (j *ProcessImageJob) Handle() error {
    // Download from storage
    imageData, err := storage.Get(j.FilePath)
    if err != nil {
        return err
    }

    // Process image (resize, watermark, etc.)
    processed := processImage(imageData)

    // Upload processed version
    processedPath := "processed/" + filepath.Base(j.FilePath)
    return storage.Put(processedPath, processed)
}
```

### With Database

```go
type File struct {
    ID        int64
    UserID    int64
    Path      string
    Size      int64
    MimeType  string
    CreatedAt time.Time
}

func SaveFile(userID int64, file io.Reader, filename string) (*File, error) {
    // Generate path
    uniqueName := storage.GenerateUniqueName(filename)
    path := fmt.Sprintf("uploads/%d/%s", userID, uniqueName)

    // Upload to storage
    if err := storage.PutFile(path, file); err != nil {
        return nil, err
    }

    // Get file info
    size, _ := storage.Size(path)

    // Save to database
    record := &File{
        UserID:    userID,
        Path:      path,
        Size:      size,
        CreatedAt: time.Now(),
    }

    db.Create(record)

    return record, nil
}
```

## Configuration

### Environment Variables

```env
STORAGE_DRIVER=local
STORAGE_LOCAL_PATH=/var/www/uploads
STORAGE_BASE_URL=https://cdn.myapp.com
```

### Initialization

```go
func InitStorage() (storage.Storage, error) {
    driver := os.Getenv("STORAGE_DRIVER")

    switch driver {
    case "local":
        basePath := os.Getenv("STORAGE_LOCAL_PATH")
        baseURL := os.Getenv("STORAGE_BASE_URL")

        storage, err := storage.NewLocalStorage(basePath, logger)
        if err != nil {
            return nil, err
        }

        storage.SetBaseURL(baseURL)
        return storage, nil

    default:
        return nil, fmt.Errorf("unknown storage driver: %s", driver)
    }
}
```

## Troubleshooting

### Permission Denied

```bash
# Ensure upload directory is writable
sudo chown -R www-data:www-data /var/www/uploads
sudo chmod -R 755 /var/www/uploads
```

### Large File Uploads

```go
// Increase upload limit
r.ParseMultipartForm(100 << 20) // 100 MB

// Use streaming for very large files
storage.PutFile(path, file) // Don't load into memory
```

### Path Issues

```bash
# Always use absolute paths
storage, _ := storage.NewLocalStorage("/var/www/uploads", logger)

# Not relative paths
storage, _ := storage.NewLocalStorage("./uploads", logger) // ❌ May fail
```
