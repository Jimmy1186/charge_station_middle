# 🔌 Chargestation Core Service

這個專案是充電站的核心服務，負責處理充電站的資料、通訊和業務邏輯。

🛠️ 開發環境設定 (Development Setup)
前提條件 (Prerequisites)
Go 語言環境: 確保您的機器上安裝了 Go (建議版本 Go 1.20+)。

配置檔案: 專案運行需要一個 config.yaml 檔案。請參考 config.example.yaml 建立您的本地配置。

運行與除錯 (Run & Debug)
您可以使用以下指令在本地環境中運行和除錯應用程式：

安裝依賴:

Bash

go mod tidy
啟動應用程式:

Bash

go run .
應用程式將讀取當前目錄下的 config.yaml 檔案，並依據配置啟動服務。

⚙️ 配置檔案 (Configuration)
本服務使用 YAML 格式的配置檔案 (config.yaml)。所有配置都透過 Viper 函式庫加載，並支援環境變數覆寫。

請確保您的 config.yaml 檔案結構符合預期，例如：

YAML

server:
port: 8080
database:
url: "postgres://user:password@host:port/dbname"
stations:

- id: "01"
  ip: "127.0.0.1"
  port: 8000
- id: "02"
  ip: "127.0.0.1"
  port: 8001
  🚀 生產環境部署 (Production Deployment)
  為了在生產環境中獲得最佳的效能和穩定性，我們採用靜態編譯的方式產生一個獨立的可執行檔。

1. 編譯可執行檔 (Build Binary)
   請在開發機器上執行以下交叉編譯指令，為目標 Linux 系統生成可執行檔：

Bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o chargestationcore ./
CGO_ENABLED=0: 確保靜態連結，不依賴目標系統的 C 函式庫。

GOOS=linux: 指定目標作業系統為 Linux。

GOARCH=amd64: 指定目標架構為 64 位元。

-o chargestationcore: 輸出檔案名稱為 chargestationcore。

./: 指定當前目錄為編譯的入口點。

2. 部署到目標伺服器 (Deploy to Server)
   將以下兩個檔案複製到您的生產伺服器上的一個資料夾中（例如 /opt/chargestation/）：

編譯好的可執行檔：chargestationcore

生產配置檔案：config.yaml

3. 運行服務 (Execute Service)
   進入目標資料夾，直接運行可執行檔：

Bash

/opt/chargestation/chargestationcore
⚠️ 建議事項: 在實際生產環境中，請使用 Systemd 或 Supervisor 等服務管理工具來運行 chargestationcore，以確保服務在崩潰時能自動重啟，並在後台持續運行。
