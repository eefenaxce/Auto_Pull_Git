# 获取所有支持的 GOOS 和 GOARCH 组合
$goos_list = @("windows", "linux", "darwin")
$goarch_list = @("amd64", "arm64")

# 创建 bin 目录（如果不存在）
if (-not (Test-Path -Path ".\bin")) {
    New-Item -ItemType Directory -Path ".\bin"
}

foreach ($goos in $goos_list) {
    foreach ($goarch in $goarch_list) {
        $env:GOOS = $goos
        $env:GOARCH = $goarch
        
        $output_name = "auto_pull_git"
        
        Write-Host "Building for $goos/$goarch..."
        if ($goos -eq "windows") {
            go build -o ".\bin\$($output_name)_$($goos)_$($goarch).exe"
        } else {
            go build -o ".\bin\$($output_name)_$($goos)_$($goarch)"
        }
        
        if ($LASTEXITCODE -ne 0) {
            Write-Host "Error building for $goos/$goarch" -ForegroundColor Red
        } else {
            Write-Host "Successfully built for $goos/$goarch" -ForegroundColor Green
        }
    }
}

Write-Host "Build process completed."