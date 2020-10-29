package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
)

var (
	DB *sql.DB
)

type Block struct {
	Timestamp int64  `db:"created_on"`
	Data      string `db:"data"`
	PrevHash  []byte `db:"prevHash"`
	Hash      []byte `db:"hash"`
}

func NewBlock(data string, prevHash []byte) Block {
	block := &Block{time.Now().Unix(), data, prevHash, []byte{}}
	return block.SetHash()
}

func (b Block) SetHash() Block {
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevHash, []byte(b.Data), timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
	time.Sleep(time.Second)
	return b
}

func Healthcheck() bool {
	return true
}

func ParseJSON(r *http.Request, v interface{}) error {
	if r == nil || r.Body == nil {
		return fmt.Errorf("No Body")
	}

	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func initDB() {
	if _, err := DB.Query("DROP TABLE IF EXISTS blockchain;"); err != nil {
		log.Println("error dropping:", err)
	}
	if _, err := DB.Query(`CREATE TABLE blockchain (
		id SERIAL PRIMARY KEY UNIQUE,
		created_on serial NOT NULL,
		data varchar,
		prevHash BYTEA UNIQUE,
		hash BYTEA UNIQUE NOT NULL)`); err != nil {
		log.Println("error creating:", err)
	}
	b := NewBlock("Genesis Block", []byte{})
	DB.QueryRow("INSERT INTO blockchain(created_on,data,hash) VALUES($1,$2,$3)",
		b.Timestamp, b.Data, b.Hash)
}

func GetFileContent(fileName string) (string, error) {
	file, err := os.Open(fileName) // O_RDONLY mode
	if err != nil {
		return "", err
	}
	defer file.Close()

	res, err := ioutil.ReadAll(file)
	return string(res), err
}

func AddBlock(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, api_key, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "Authorization")
	data := struct {
		Data string `json:"data"`
	}{}
	if err := ParseJSON(r, &data); err != nil {
		fmt.Fprintln(w, err)
		return
	}

	var prevHash []byte
	err := DB.QueryRow(`SELECT "hash"
	FROM "blockchain"
	ORDER BY "id" DESC
	LIMIT 1`).Scan(&prevHash,)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	newBlock := NewBlock(data.Data, prevHash)

	DB.QueryRow("INSERT INTO blockchain(created_on,data,prevhash,hash) VALUES($1,$2,$3,$4)",
		newBlock.Timestamp, newBlock.Data, newBlock.PrevHash, newBlock.Hash,
	)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, "{\"done\": \"true\"}")
}

func DisplayBlockChain(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	rows, err := DB.Query(`SELECT "created_on","data","prevhash","hash" FROM "blockchain";`)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	var bc []Block
	for rows.Next() {
		var b Block
		if err := rows.Scan(&b.Timestamp, &b.Data, &b.PrevHash, &b.Hash); err != nil {
			continue
		}
		bc = append(bc, b)
	}

	fileContent, err := GetFileContent("/blockchain.html")
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	t, err := template.New("webpage").Parse(fileContent)
	if err != nil {
		fmt.Fprintln(w, err)
	}
	if err := t.Execute(w, bc); err != nil {
		log.Println(err)
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--healthcheck" {
		if Healthcheck() {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	HTTP_IP := os.Getenv("IP")
	HTTP_PORT := os.Getenv("PORT")
	DB_HOST := "postgresql"
	DB_USER := "postgres"
	DB_PASS := os.Getenv("POSTGRES_PASSWORD")

	dbinfo := fmt.Sprintf("host=%s user=%s password=%s sslmode=disable",
		DB_HOST, DB_USER, DB_PASS)
	var err error
	DB, err = sql.Open("postgres", dbinfo)
	if err != nil {
		log.Fatal(err)
	}
	defer DB.Close()

	initDB()

	router := httprouter.New()
	router.GET("/", DisplayBlockChain)
	router.POST("/add", AddBlock)

	log.Printf("Starting server at http://%s:%s\n", HTTP_IP, HTTP_PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%s", HTTP_IP, HTTP_PORT), router))
}
