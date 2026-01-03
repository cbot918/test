package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

func main() {
	serverAddr := "fss.twcos.com:5000"
	fmt.Printf("正在連線到 %s ...\n", serverAddr)
	fmt.Println("------------------------------------------------")
	fmt.Println("★ 操作提示：")
	fmt.Println("1. 正常輸入文字按 Enter 發送")
	fmt.Println("2. 若需發送 [Ctrl+C] 訊號，請輸入 #c 然後按 Enter")
	fmt.Println("3. 若需發送 [Ctrl+U] 訊號，請輸入 #u 然後按 Enter")
	fmt.Println("------------------------------------------------")

	// 建立 TCP 連線
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("連線失敗: %v\n", err)
		return
	}
	defer conn.Close()

	// --- 接收端 (Goroutine) ---
	// 負責從 Server 讀取 Big5 資料 -> 轉成 UTF-8 -> 印在螢幕上
	go func() {
		// 建立解碼器 Reader (Big5 -> UTF-8)
		decoderReader := transform.NewReader(conn, traditionalchinese.Big5.NewDecoder())

		// 使用 io.Copy 串流輸出到螢幕
		// 這樣即使 Server 傳來沒有換行的提示字串 (如 "Login:") 也能即時顯示
		_, err := io.Copy(os.Stdout, decoderReader)
		if err != nil {
			// 當 Server 斷開連線或網路錯誤時會執行到這裡
			fmt.Println("\n\n[連線已中斷]")
			os.Exit(0) // 直接結束程式
		}
	}()

	// --- 發送端 (主程式) ---
	// 負責讀取你的鍵盤輸入 (UTF-8) -> 判斷特殊指令 -> 轉成 Big5 -> 送給 Server
	encoderWriter := transform.NewWriter(conn, traditionalchinese.Big5.NewEncoder())
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		inputText := scanner.Text()

		var dataToSend []byte

		// 判斷是否為特殊指令
		switch inputText {
		case "#c":
			fmt.Println(">> 發送 Ctrl+C 訊號...")
			// \x03 是 Ctrl+C 的 ASCII 碼，後面補上 \r\n 代表 Enter
			dataToSend = []byte{'\x03', '\r', '\n'}
		case "#u":
			fmt.Println(">> 發送 Ctrl+U 訊號...")
			// \x15 是 Ctrl+U 的 ASCII 碼 (通常用來刪除整行)
			dataToSend = []byte{'\x15', '\r', '\n'}
		default:
			// 正常文字輸入，補上 CRLF 換行符號
			msg := inputText + "\r\n"
			dataToSend = []byte(msg)
		}

		// 寫入資料 (Writer 會自動將字串轉為 Big5 編碼送出)
		if _, err := encoderWriter.Write(dataToSend); err != nil {
			fmt.Printf("發送失敗: %v\n", err)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("鍵盤讀取錯誤: %v\n", err)
	}
}
