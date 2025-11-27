# 📥 Installing Go

The Go SDK is ready to publish, but you need to install Go first.

---

## ✅ Quick Install (Windows)

### Option 1: Using Chocolatey (Recommended)

```powershell
# Install Chocolatey if not already installed
# Visit: https://chocolatey.org/install

# Install Go
choco install golang

# Verify installation
go version
```

### Option 2: Manual Install

1. Go to: https://go.dev/dl/
2. Download the Windows installer (`.msi`)
3. Run the installer
4. Restart PowerShell
5. Verify: `go version`

---

## 🔧 After Installation

Once Go is installed, you can:

### 1. Test the SDK locally
```bash
cd sdk/go
go mod tidy
go run examples/main.go
```

### 2. Publish to pkg.go.dev
```bash
cd sdk/go
git init
git add .
git commit -m "Release v1.0.0"
git tag v1.0.0
git remote add origin https://github.com/wowsql/wowsql-go.git
git push origin main v1.0.0
```

---

## 📚 Resources

- **Go Downloads**: https://go.dev/dl/
- **Go Documentation**: https://go.dev/doc/
- **Go Tutorial**: https://go.dev/tour/

---

**The Go SDK is complete and ready - just install Go to publish it!** ✅

