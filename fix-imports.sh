#!/bin/bash
# Fix all incorrect import paths

echo "Fixing import paths..."

# Use PowerShell since we're on Windows
powershell -Command "
Get-ChildItem -Path . -Include '*.go' -Recurse | ForEach-Object {
    \$content = Get-Content \$_.FullName -Raw
    if (\$content -match 'github\.com/storage-system') {
        Write-Host \"Fixing: \$(\$_.FullName)\"
        \$content = \$content -replace 'github\.com/storage-system', 'storage-engine'
        Set-Content -Path \$_.FullName -Value \$content -NoNewline
    }
}
"

echo "Import paths fixed!"
