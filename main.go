package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/tidwall/gjson"
	"golang.org/x/sync/errgroup"
)

const (
	URLTemplate      = "https://gxh.vip.qq.com/qqshow/admindata/comdata/vipEmoji_item_%d/xydata.js"
	ConcurrencyLimit = 25
	FromID           = 213443
)

var NotFoundErr = fmt.Errorf("not found")

func getEmojiData(id int) error {
	url := fmt.Sprintf(URLTemplate, id)
	path := fmt.Sprintf("data/%d.js", id)

	// see if the file already exists
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return NotFoundErr
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, data, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func download() {
	os.MkdirAll("data", os.ModePerm)

	limiter := make(chan struct{}, ConcurrencyLimit)
	errs := errgroup.Group{}
	// fetch from FromID and all the way until there's a 404 error
	done := false
	for i := FromID; ; i++ {
		limiter <- struct{}{}

		if done {
			break
		}
		id := i
		errs.Go(func() error {
			defer func() { <-limiter }()
			fmt.Println("fetching", id)
			err := getEmojiData(id)
			if err != nil {
				// done = true
				fmt.Println("error:", err)
				return nil
			}
			return err
		})
	}

	err := errs.Wait()
	if err != nil {
		panic(err)
	}
}

var FilePaths = `data/207070.js
data/205281.js
data/231500.js
data/233613.js
data/231435.js
data/204640.js
data/203469.js
data/205242.js
data/231561.js
data/232499.js
data/203391.js
data/231504.js
data/233821.js
data/231520.js
data/205004.js
data/205284.js
data/205061.js
data/208405.js
data/233707.js
data/231560.js
data/206976.js
data/204705.js
data/231521.js
data/208666.js
data/207222.js
data/231501.js
data/231606.js
data/208672.js
data/208870.js
data/231540.js
data/231298.js
data/209583.js
data/205782.js
data/209626.js
data/231330.js
data/203764.js
data/207188.js
data/233013.js
data/231334.js
data/205089.js
data/203857.js
data/209535.js
data/231759.js
data/231499.js
data/232752.js
data/231748.js
data/206154.js
data/231335.js
data/206352.js
data/209118.js
data/232713.js
data/209571.js
data/206372.js
data/208293.js
data/204020.js
data/231383.js
data/231768.js
data/231538.js
data/233050.js
data/209610.js
data/231508.js
data/231332.js
data/232239.js
data/204235.js
data/232714.js
data/205510.js
data/232536.js
data/231518.js
data/231509.js
data/231558.js
data/207865.js
data/208165.js
data/231333.js
data/233385.js
data/203470.js
data/205148.js
data/208246.js
data/231519.js
data/231539.js
data/206296.js
data/231243.js
data/208070.js
data/231366.js
data/231337.js
data/205129.js
data/207399.js
data/208145.js
data/231563.js
data/204500.js
data/207738.js
data/208406.js
data/231506.js
data/231502.js
data/209578.js
data/204811.js
data/233734.js
data/206732.js
data/208692.js
data/231437.js
data/209030.js
data/231368.js
data/208584.js
data/231503.js
data/231566.js
data/231537.js
data/206993.js
data/231436.js
data/232051.js
data/231562.js
data/231447.js
data/231507.js
data/231517.js
data/231546.js
data/231894.js
data/231432.js
data/209306.js`

func main() {
	filePaths := strings.Split(FilePaths, "\n")
	fmt.Println(filePaths)
	wg := sync.WaitGroup{}
	limiter := make(chan struct{}, 20)

	for _, path := range filePaths {
		file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			panic(err)
		}
		data, err := io.ReadAll(file)
		if err != nil {
			panic(err)
		}
		str := string(data)
		str = str[strings.Index(str, "{"):]
		j := gjson.Parse(str)
		d := j.Get("data")
		packName := d.Get("baseInfo").Array()[0].Get("name").String()

		emojis := d.Get("md5Info").Array()
		if len(emojis) == 0 {
			fmt.Println("no emojis in", packName)
			continue
		}
		for _, emoji := range emojis {
			limiter <- struct{}{}
			wg.Add(1)

			go func(emoji gjson.Result) {
				defer wg.Done()

				name := emoji.Get("name").String()
				md5 := emoji.Get("md5").String()
				url := fmt.Sprintf("https://i.gtimg.cn/club/item/parcel/item/%s/%s/300x300.png", md5[:2], md5)
				resp, err := http.Get(url)
				if err != nil {
					panic(err)
				}

				if resp.StatusCode != http.StatusOK {
					fmt.Println("not found:", url)
				}

				// save to downloaded/packName/name.png
				fmt.Println("saving", packName, name)

				os.MkdirAll(fmt.Sprintf("downloaded/%s", packName), os.ModePerm)

				f, _ := os.OpenFile(fmt.Sprintf("downloaded/%s/%s.png", packName, name), os.O_CREATE|os.O_WRONLY, os.ModePerm)
				io.Copy(f, resp.Body)

				f.Close()
				resp.Body.Close()

				<-limiter
			}(emoji)
		}
	}
}
