package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

func main() {
	serverAddr := "fss.twcos.com:5000"
	fmt.Printf("正在連線到 %s ...\n", serverAddr)
	fmt.Println("------------------------------------------------")
	fmt.Println("★ 功能說明：")
	fmt.Println("1. 系統會自動每 5 分鐘發送一次 'l' 指令")
	fmt.Println("2. 手動輸入文字按 Enter 可正常發送")
	fmt.Println("3. 輸入 #c 可發送 [Ctrl+C] 訊號")
	fmt.Println("------------------------------------------------")

	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("連線失敗: %v\n", err)
		return
	}
	defer conn.Close()

	// --- 核心修改：建立一個「發送通道」 ---
	// 所有要送給 Server 的資料（不論是手打的還是自動的），都丟進這裡
	sendChan := make(chan []byte)

	// --- 1. 寫入專用 Goroutine (唯一的發送者) ---
	// 只有這個 func 會真正執行 Write，避免多個執行緒同時寫入造成錯亂
	go func() {
		// 建立 Big5 編碼器
		encoderWriter := transform.NewWriter(conn, traditionalchinese.Big5.NewEncoder())

		// 一直從通道拿資料出來發送
		for data := range sendChan {
			_, err := encoderWriter.Write(data)
			if err != nil {
				fmt.Printf("寫入失敗: %v\n", err)
				return
			}
		}
	}()

	// --- 2. 自動排程 Goroutine (每 5 分鐘) ---
	go func() {
		// 設定計時器：5 分鐘
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			// 時間到！把指令丟進通道
			// fmt.Println(">> [自動發送] l 指令...") // 如果覺得太吵可以把這行註解掉
			sendChan <- []byte("hp\r\n")
		}
	}()

	// --- 3. 接收專用 Goroutine (負責聽並印出) ---
	go func() {
		decoderReader := transform.NewReader(conn, traditionalchinese.Big5.NewDecoder())
		_, err := io.Copy(os.Stdout, decoderReader)
		if err != nil {
			fmt.Println("\n\n[連線已中斷]")
			os.Exit(0)
		}
	}()

	// --- 4. 主程式 (處理鍵盤輸入) ---
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		inputText := scanner.Text()
		var dataToSend []byte

		switch inputText {
		case "#c":
			fmt.Println(">> [手動發送] Ctrl+C 訊號")
			dataToSend = []byte{'\x03', '\r', '\n'}
		case "#u":
			fmt.Println(">> [手動發送] Ctrl+U 訊號")
			dataToSend = []byte{'\x15', '\r', '\n'}

		case "quit":
			fmt.Println(">> [手動發送] quit 指令")
			sendChan <- dataToSend
			conn.Close()
			os.Exit(0)
		default:
			// 正常文字，補上換行
			dataToSend = []byte(inputText + "\r\n")
		}

		// 把資料丟進通道，讓寫入專用 Goroutine 幫我們送
		sendChan <- dataToSend
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("鍵盤讀取錯誤: %v\n", err)
	}
}
