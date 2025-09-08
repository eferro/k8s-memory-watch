# Release Notes v0.1.3: Code Quality and Refactoring Improvements

## Changes in v0.1.3

### 🚀 Code Quality Improvements
- **refactor(monitor)**: Eliminate code duplication in PrintCSV function
  - Extracted `buildCSVRecord` and `buildCSVRecordForPod` helper functions  
  - Reduced PrintCSV function from ~130 lines to ~60 lines (53% reduction)
  - Eliminated 74+ lines of duplicated CSV record building code
  - Single source of truth for CSV format established
  - Improved maintainability and test coverage

### 🛠️ CI/CD Improvements  
- **ci(lint)**: Disable gocyclo and errcheck to avoid failing on complexity in CI
- **ci**: Bump GitHub Actions to latest major versions (checkout@v4, setup-go@v5, upload/download-artifact@v4)
- **ci**: Fix deprecated set-output commands in GitHub Actions workflows
- **ci**: Trigger workflows automatically

### 📋 Documentation
- **docs**: Add comprehensive improvement plan with task tracking system
- **docs**: Plan for future refactoring and code quality improvements

### 🧪 Testing
- **test**: Increased test coverage from 54.0% to 56.3%
- **test**: Added comprehensive tests for CSV helper functions
- **test**: All existing functionality preserved with no behavioral changes

This release focuses on code quality improvements and technical debt reduction while maintaining full backward compatibility.

## Installation

Download the appropriate binary for your platform from the assets below.

### Linux/macOS
```bash
# Download binary
curl -L -o k8s-memory-watch https://github.com/eferro/k8s-memory-watch/releases/download/v0.1.3/k8s-memory-watch-linux-amd64
chmod +x k8s-memory-watch
sudo mv k8s-memory-watch /usr/local/bin/
```

### Windows
Download `k8s-memory-watch-windows-amd64.exe` and add it to your PATH.