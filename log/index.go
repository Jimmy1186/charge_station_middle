package log

import (
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"time" // <-- 導入 time 套件

	"gopkg.in/natefinch/lumberjack.v2"
)

// InitLog 初始化日誌系統，使用日期作為日誌檔案名稱的一部分
func InitLog() {

    kenmecPath, err := GetKenmecFilePath()
    if err != nil {
        fmt.Println("錯誤: 無法獲取 Kenmec 路徑:", err)
        return
    }
    
    // 1. 定義日誌目錄 (e.g., C:\Users\username\kenmec\_logs)
    logDir := filepath.Join(kenmecPath, "_logs/charge_station")

    // 2. 檢查目錄是否存在，若不存在則創建
    if err := os.MkdirAll(logDir, 0755); err != nil {
        fmt.Printf("錯誤: 無法創建日誌目錄 %s: %v\n", logDir, err)
        return 
    }
    
    // 3. 動態構建檔案名稱：charge_station_YYYY-MM-DD.log
    // time.Now().Format("2006-01-02") 會輸出當前日期，格式為 YYYY-MM-DD
    dateStr := time.Now().Format("2006-01-02") 
    baseName := fmt.Sprintf("charge_station_%s.log", dateStr)
    
    // 4. 組合完整的日誌檔案路徑
    logFileName := filepath.Join(logDir, baseName) 
    
    // 5. 設定 Lumberjack 輪替規則
    logRotator := &lumberjack.Logger{
        // Filename 現在是帶有當天日期的完整路徑
        Filename:   logFileName, 
        MaxSize:    100,       // 檔案超過 100MB 進行切割
        MaxBackups: 5,         // 最多保留 5 個備份檔案
        MaxAge:     30,        // 最多保留 30 天的日誌
        Compress:   true,      // 輪替時進行 Gzip 壓縮
    }

    // 6. 建立 slog 的 Handler
    multiWriter := slog.NewJSONHandler(logRotator, nil)

    // 7. 設定為預設 Logger
    logger := slog.New(multiWriter)
    slog.SetDefault(logger)

    slog.Info("日誌系統初始化成功", "log_path", logFileName)
}

// GetKenmecFilePath 保持不變
func GetKenmecFilePath() (string, error) {
    currentUser, err := user.Current()
    if err != nil {
        return "", fmt.Errorf("無法獲取用戶主目錄: %w", err)
    }
    homeDir := currentUser.HomeDir

    kenmecPath := filepath.Join(homeDir, "kenmec")

    return kenmecPath, nil
}