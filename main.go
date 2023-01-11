package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/png"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type URIJson struct {
	Image string `json:"image"`
}

var (
	t    *template.Template
	bayc *Store
)

func main() {

	var err error

	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)

	log.Printf("APP Started")
	log.Printf("PORT:%s", os.Getenv("PORT"))
	http.HandleFunc("/random", RenderRandom)

	t, err = template.ParseFiles("./template.tpl")
	if err != nil {
		log.Printf("ParseFiles - Err:%s", err)
		return
	}

	ethConn, err := ethclient.Dial("https://mainnet.infura.io/v3/a20c03429ae14156bcc7b16299668383")
	if err != nil {
		log.Printf("Dial - Err:%s", err)
		return
	}

	bayc, err = NewStore(common.HexToAddress("0xb363af6181a4335608880510772a5f61a5183c88"), ethConn)
	if err != nil {
		log.Fatal(err)
	}

	err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func RenderRandom(w http.ResponseWriter, req *http.Request) {

	params := req.URL.Query()
	refresh := 30
	if params.Get("refresh") != "" {
		refreshTmp, err := strconv.Atoi(params.Get("refresh"))
		if err == nil {
			refresh = refreshTmp
		}
	}

	tokenID, img, err := GetRandom()
	if err != nil {
		log.Printf("GetRandom - Err:%s", err)
		time.Sleep(time.Second * 10)
		http.Redirect(w, req, "/random", http.StatusTemporaryRedirect)
		return
	} else {
		resp, err := http.Get(img)
		if err != nil {
			log.Printf("Get - Err:%s", err)
			time.Sleep(time.Second * 10)
			http.Redirect(w, req, "/random", http.StatusTemporaryRedirect)
			return
		}
		defer resp.Body.Close()

		imgDecoded, _, err := image.Decode(resp.Body)
		if err != nil {
			log.Printf("Decode - Err:%s", err)
			time.Sleep(time.Second * 10)
			http.Redirect(w, req, "/random", http.StatusTemporaryRedirect)
			return
		}

		r, g, b, a := imgDecoded.At(100, 100).RGBA()

		owner, err := bayc.OwnerOf(&bind.CallOpts{}, big.NewInt(tokenID))
		if err != nil {
			log.Printf("Failed to retrieve token name: %v", err)
			http.Redirect(w, req, "/random", http.StatusTemporaryRedirect)
			return
		}
		fmt.Println("owner:", owner)

		data := struct {
			Refresh int
			Img     string
			R       uint32
			G       uint32
			B       uint32
			A       uint32
		}{
			refresh,
			img,
			r / 257,
			g / 257,
			b / 257,
			a / 257,
		}

		err = t.Execute(w, data)
		if err != nil {
			log.Printf("Execute - Err:%s", err)
			time.Sleep(time.Second * 10)
			http.Redirect(w, req, "/random", http.StatusTemporaryRedirect)
			return
		}
	}
}

func GetRandom() (int64, string, error) {
	rand.Seed(time.Now().Unix())
	r := rand.Intn(4003)

	uri := fmt.Sprintf("https://ipfs.sloppyta.co/ipfs/QmaTod6T4zdi77pozmHonriDsGnSBEga48N45hqaW3KPQT/%d.json", r)

	log.Printf("uri:%v", uri)

	resp, err := http.Get(uri)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	var uriJson URIJson

	err = json.NewDecoder(resp.Body).Decode(&uriJson)
	if err != nil {
		return 0, "", err
	}

	log.Printf("uriJson:%v", uriJson)

	return int64(r), strings.Replace(uriJson.Image, "ipfs:/", "https://ipfs.sloppyta.co/ipfs/", -1), nil
}
