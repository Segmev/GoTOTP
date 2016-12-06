package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/andlabs/ui"
)

type TitleKey struct {
	Title string `json:"title"`
	Key   []byte `json:"key"`
}

func toBytes(value int64) []byte {
	var result []byte
	mask := int64(0xFF)
	shifts := [8]uint16{56, 48, 40, 32, 24, 16, 8, 0}
	for _, shift := range shifts {
		result = append(result, byte((value>>shift)&mask))
	}
	return result
}

func toUint32(bytes []byte) uint32 {
	return (uint32(bytes[0]) << 24) +
		(uint32(bytes[1]) << 16) +
		(uint32(bytes[2]) << 8) +
		uint32(bytes[3])
}

func OTP(key []byte, value []byte) uint32 {
	hmacSha1 := hmac.New(sha1.New, key)
	hmacSha1.Write(value)
	hash := hmacSha1.Sum(nil)
	offset := hash[len(hash)-1] & 0x0F
	hashParts := hash[offset : offset+4]
	hashParts[0] = hashParts[0] & 0x7F
	number := toUint32(hashParts)
	pwd := number % 1000000
	return pwd
}

func cleanKey(input string) ([]byte, error) {
	inputNoSpaces := strings.Replace(input, " ", "", -1)
	inputNoSpacesUpper := strings.ToUpper(inputNoSpaces)
	return (base32.StdEncoding.DecodeString(inputNoSpacesUpper))
}

var mut sync.Mutex
var entriesb []*ui.Entry
var keys []([]byte)
var boxes []*ui.Box
var titles []string
var ToEncode []TitleKey

func saveKeys(errorb *ui.Label) {
	mut.Lock()
	ToEncode = ToEncode[:0]
	for i := range keys {
		ToEncode = append(ToEncode, TitleKey{Title: titles[i], Key: keys[i]})
	}
	mut.Unlock()
	b, err := json.Marshal(ToEncode)
	if err != nil {
		errorb.SetText("Can't save keys.")
		go func() {
			time.Sleep(time.Second * 5)
			errorb.SetText("")
		}()
		return
	}
	err = ioutil.WriteFile("saved", b, 0644)
	if err != nil {
		errorb.SetText("Can't save keys.")
		go func() {
			time.Sleep(time.Second * 5)
			errorb.SetText("")
		}()
	}
}

func loadKeys(box2 *ui.Box, entrytitle *ui.Entry) {
	lignebox := ui.NewHorizontalBox()
	lignebox.Append(ui.NewLabel("  Label"), true)
	lignebox.Append(ui.NewLabel("  OTP"), true)
	boxes = append(boxes, lignebox)
	box2.Append(lignebox, true)
	data, err := ioutil.ReadFile("saved")
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &ToEncode)
	for i := range ToEncode {
		keys = append(keys, ToEncode[i].Key)
		fillEntry(box2, ToEncode[i].Title)
	}
}

func fillEntry(box2 *ui.Box, entrytitle string) {
	mut.Lock()
	entriesb = append(entriesb, ui.NewEntry())
	lignebox := ui.NewHorizontalBox()
	titlelabel := ui.NewEntry()
	if entrytitle != "" {
		titlelabel.SetText(fmt.Sprintf("key: %s", entrytitle))
	} else {
		titlelabel.SetText(fmt.Sprintf("key: %d", len(entriesb)))
	}
	lignebox.Append(titlelabel, true)
	lignebox.Append(entriesb[len(entriesb)-1], true)
	boxes = append(boxes, lignebox)
	box2.Append(lignebox, true)
	titles = append(titles, entrytitle)
	mut.Unlock()
}

func loadWind() {
	err := ui.Main(func() {
		entrykey := ui.NewEntry()
		entrytitle := ui.NewEntry()
		button := ui.NewButton("Add key")
		errorbox := ui.NewLabel("")
		remainingTime := ui.NewLabel("")
		progressBar := ui.NewProgressBar()
		box := ui.NewVerticalBox()
		button2 := ui.NewButton("Delete last entry")
		box2 := ui.NewVerticalBox()
		button2.OnClicked(func(arg1 *ui.Button) {
			if len(entriesb) > 0 {
				mut.Lock()
				box2.Delete(len(entriesb) - 1)
				entriesb = append(entriesb[:len(entriesb)-1])
				keys = append(keys[:len(keys)-1])
				mut.Unlock()
			}
		})
		savebutton := ui.NewButton("Save Keys")
		savebutton.OnClicked(func(arg1 *ui.Button) {
			saveKeys(errorbox)
		})
		deleteSaveButton := ui.NewButton("Delete saved keys")
		deleteSaveButton.OnClicked(func(arg1 *ui.Button) {
			os.Remove("saved")
		})
		labelentrybox := ui.NewHorizontalBox()
		labelentrybox.SetPadded(true)
		entrybox := ui.NewHorizontalBox()
		entrybox.SetPadded(true)
		labelentrybox.Append(ui.NewLabel("Key:"), true)
		labelentrybox.Append(ui.NewLabel("Title:"), true)
		entrybox.Append(entrykey, true)
		entrybox.Append(entrytitle, true)
		box.Append(labelentrybox, false)
		box.Append(entrybox, false)
		addDelBox := ui.NewHorizontalBox()
		addDelBox.Append(button, true)
		addDelBox.Append(button2, true)
		addDelBox.SetPadded(true)
		box.Append(addDelBox, false)
		saveDelBox := ui.NewHorizontalBox()
		saveDelBox.SetPadded(true)
		saveDelBox.Append(savebutton, true)
		saveDelBox.Append(deleteSaveButton, true)
		box.Append(saveDelBox, false)
		box.Append(remainingTime, false)
		box.Append(progressBar, false)
		box.Append(errorbox, false)
		box.Append(ui.NewHorizontalSeparator(), false)
		box.Append(box2, false)
		box.SetPadded(true)
		ckey := make(chan []byte)
		loadKeys(box2, entrytitle)
		go func(ckey <-chan []byte) {
			for {
				epochSeconds := time.Now().Unix()
				secondsRemaining := 30 - (epochSeconds % 30)
				for epochSeconds%30 != 0 {
					secondsRemaining = 30 - (epochSeconds % 30)
					if len(keys) > 0 {
						progressBar.SetValue(int(float64(epochSeconds%30)*3.35) % 100)
						remainingTime.SetText(fmt.Sprintf("%d second(s) remaining", secondsRemaining))
					}
					time.Sleep(time.Second)
					epochSeconds = time.Now().Unix()
					state := true
					for state {
						select {
						case key := <-ckey:
							if key != nil {
								keys = append(keys, key)
							}
						default:
							state = false
						}
					}
					mut.Lock()
					for i := range keys {
						otp := OTP(keys[i], toBytes(epochSeconds/30))
						if titles[i] == "" {
							entriesb[i].SetText(fmt.Sprintf("%.06d", otp))
						} else {
							entriesb[i].SetText(fmt.Sprintf("%.06d\t", otp))
						}
					}
					mut.Unlock()
				}
			}
		}(ckey)
		window := ui.NewWindow("Google OTP generator", 500, 200, false)
		window.SetChild(box)
		window.SetMargined(true)
		button.OnClicked(func(*ui.Button) {
			key, err := cleanKey(entrykey.Text())
			//print(key)
			if err == nil && len(key) > 0 {
				errorbox.SetText("")
				fillEntry(box2, entrytitle.Text())
				go func(key []byte, ckey chan []byte) {
					ckey <- key
				}(key, ckey)
			} else {
				errorbox.SetText("Invalid key: Not a compatible Google secret key.")
				go func() {
					time.Sleep(time.Second * 5)
					errorbox.SetText("")
				}()
			}
		})
		window.OnClosing(func(*ui.Window) bool {
			ui.Quit()
			os.Exit(0)
			return true
		})
		window.Show()
	})
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-w" || os.Args[1] == "--window") {
		loadWind()
	} else if len(os.Args) > 1 && !(os.Args[1] == "-w" || os.Args[1] == "--window") {
		key, err := cleanKey(os.Args[1])
		if err != nil {
			fmt.Println("Error: Not a compatible Google secret key")
			os.Exit(1)
		}
		for {
			epochSeconds := time.Now().Unix()
			secondsRemaining := 30 - (epochSeconds % 30)
			for epochSeconds%30 != 0 {
				secondsRemaining = 30 - (epochSeconds % 30)
				fmt.Printf("\rkey: %.06d   (remaining time : %d)", OTP(key, toBytes(time.Now().Unix()/30)), secondsRemaining)
				time.Sleep(time.Second)
				epochSeconds = time.Now().Unix()
			}
		}
	} else {
		loadWind()
	}
}
