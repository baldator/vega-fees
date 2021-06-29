package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/binance-exchange/go-binance"
	"github.com/go-kit/kit/log"
	"github.com/go-pg/pg/v10"
)

const binanceURL = "https://data.binance.vision/data/spot/monthly/klines/"
const samplingPeriod = "15m"
const dataDir = "./data"
const binanceLimit = 1500

func scrapeKlines(pair *Pair, db *pg.DB) error {
	var logger log.Logger
	//	tm := time.Now().Unix()
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "time", log.DefaultTimestampUTC, "caller", log.DefaultCaller)

	hmacSigner := &binance.HmacSigner{
		Key: []byte("API secret"),
	}
	ctx, _ := context.WithCancel(context.Background())
	// use second return value for cancelling request when shutting down the app

	binanceService := binance.NewAPIService(
		"https://www.binance.com",
		"API key",
		hmacSigner,
		logger,
		ctx,
	)
	b := binance.NewBinance(binanceService)

	kl, err := b.Klines(binance.KlinesRequest{
		Symbol:    pair.Id,
		Interval:  binance.FifteenMinutes,
		StartTime: pair.LastParsedTimestamp * 1000,
		Limit:     binanceLimit,
	})
	if err != nil {
		return err
	}

	for _, kline := range kl {
		klineTmp := &Kline{
			PairId: pair.Id,
			Time:   kline.OpenTime.Unix(),
			Value:  kline.Open,
		}
		err = updateKline(klineTmp, db)

		if err != nil {
			return err
		}
	}

	// update pair
	if len(kl) > 0 {
		endTime := kl[len(kl)-1].CloseTime.Unix() + 1
		pair.LastParsedTimestamp = endTime

		err = updatePair(pair, db)
		if err != nil {
			return err
		}

	}

	return nil
}

func scrapeKlinesold(pair string, timestamp int64, db *pg.DB) error {
	yearNow, monthNowM, _ := time.Now().Date()
	tm := time.Unix(timestamp, 0)
	year := tm.Year()
	month := int(tm.Month())
	monthNow := int(monthNowM)

	for {

		url := binanceURL + pair + "/" + samplingPeriod + "/" + pair + "-" + samplingPeriod + "-" + strconv.Itoa(year) + "-" + strconv.Itoa(month) + ".zip"
		outFile := dataDir + "/" + pair + "-" + samplingPeriod + "-" + strconv.Itoa(year) + "-" + strconv.Itoa(month) + ".zip"

		// Download file
		err := DownloadFile(outFile, url)
		if err != nil {
			panic(err)
		}
		//log.Println("Downloaded: " + url)

		// Unzip file
		files, err := Unzip(outFile, dataDir)
		if err != nil {
			return err
		}

		fmt.Println("Unzipped:\n" + strings.Join(files, "\n"))

		if year == yearNow && month == monthNow {
			break
		}

		if month == 12 {
			month = 1
			year = year + 1
		} else {
			month = month + 1
		}

	}

	return nil
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
