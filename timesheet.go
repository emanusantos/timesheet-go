package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"
)

type CommitAuthor struct {
	Name  string
	Email string
	Date  string
}

type CommitStruct struct {
	Message string
	Author  CommitAuthor
}

type CommitsResponse struct {
	Commit   CommitStruct
	Html_Url string
}

func CheckError(err error) {
	if err != nil {
		fmt.Printf("Erro ao requisitar commits: %s\n", err)
		os.Exit(1)
	}
}

func GetKey(args []string) (string, error) {
	if len(args) < 2 {
		return "", errors.New("Chave do Github não informada")
	}

	return args[1], nil
}

func GetUserInput() string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Informe a data de incidência dos commits (DD/MM):")

	text, _ := reader.ReadString('\n')

	return text
}

func GetDate(text string, timestamp string) time.Time {
	layout := "2006-01-02T15:04:05.999Z"

	splittedString := strings.Split(text, "/")

	day := splittedString[0]
	month := strings.TrimSpace(splittedString[1])

	stringArr := [5]string{"2024-", month, "-", day, timestamp}

	parsedString := strings.Join(stringArr[:], "")

	date, _ := time.Parse(layout, parsedString)

	return date
}

func FormatEndpoint(endpoint, startDate, endDate string) string {
	return fmt.Sprintf(endpoint, startDate, endDate)
}

func FetchEndpoint(endpoint, key string) []byte {
	client := &http.Client{}

	request, err := http.NewRequest("GET", endpoint, nil)

	CheckError(err)

	request.Header.Add("User-Agent", "emanusantos")
	request.Header.Add("Authorization", "Bearer"+" "+key)

	response, err := client.Do(request)

	CheckError(err)

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)

	CheckError(err)

	return body
}

func ParseJsonResponse(buffer []byte) []CommitsResponse {
	var commits []CommitsResponse

	json.Unmarshal([]byte(buffer), &commits)

	return commits
}

func SortCommits(commits *[]CommitsResponse) {
	sort.Slice((*commits)[:], func(i, j int) bool {
		layout := "2006-01-02T15:04:05.999Z"
		dateI, _ := time.Parse(layout, (*commits)[i].Commit.Author.Date)
		dateJ, _ := time.Parse(layout, (*commits)[j].Commit.Author.Date)

		return dateJ.Before(dateI)
	})
}

func WriteCommitsToFile(commits []CommitsResponse) {
	os.Remove("./output.txt")

	output, err := os.Create("./output.txt")

	CheckError(err)

	var content []byte
	var links []byte

	for i := 0; i < len(commits); i++ {
		layout := "2006-01-02T15:04:05.999Z"
		date, _ := time.Parse(layout, commits[i].Commit.Author.Date)
		loc, _ := time.LoadLocation("America/Sao_Paulo")

		message := date.In(loc).Format("15:04 PM") + " - " + commits[i].Commit.Message + "\n"

		content = append(content, message...)
	}

	output.Write(content)

	output.Write([]byte("\nCommits:\n"))

	for i := 0; i < len(commits); i++ {
		commit := commits[i].Html_Url + "\n"

		links = append(links, commit...)
	}

	output.Write(links)
}

func main() {
	key, err := GetKey(os.Args)

	CheckError(err)

	text := GetUserInput()

	startDate := GetDate(text, "T00:00:00.000Z").Format("2006-01-02T15:04:05Z")
	endDate := GetDate(text, "T23:59:59.000Z").Format("2006-01-02T15:04:05Z")

	urls := [6]string{
		FormatEndpoint("https://api.github.com/repos/SavingBucks-com/SavingBucks.app/commits?author=emanusantos&sha=develop&since=%s&until=%s", startDate, endDate),
		FormatEndpoint("https://api.github.com/repos/SavingBucks-com/SavingBucks.api/commits?author=emanusantos&sha=develop&since=%s&until=%s", startDate, endDate),
		FormatEndpoint("https://api.github.com/repos/SavingBucks-com/SavingBucks.metrics/commits?author=emanusantos&sha=develop&since=%s&until=%s", startDate, endDate),
		FormatEndpoint("https://api.github.com/repos/SavingBucks-com/SavingBucks.webApp/commits?author=emanusantos&sha=develop&since=%s&until=%s", startDate, endDate),
		FormatEndpoint("https://api.github.com/repos/SavingBucks-com/SavingBucks.admin/commits?author=emanusantos&sha=develop&since=%s&until=%s", startDate, endDate),
		FormatEndpoint("https://api.github.com/repos/SavingBucks-com/SavingBucks.website/commits?author=emanusantos&sha=develop&since=%s&until=%s", startDate, endDate),
	}

	var responses []CommitsResponse
	var wg sync.WaitGroup

	wg.Add(len(urls))

	for _, url := range urls {
		go func(currentUrl string) {
			response := FetchEndpoint(currentUrl, key)

			jsonResponse := ParseJsonResponse(response)

			responses = append(responses, jsonResponse...)

			wg.Done()
		}(url)
	}

	wg.Wait()

	SortCommits(&responses)

	WriteCommitsToFile(responses)

	fmt.Println("\nCommits extraídos com sucesso.\n")

	cmd := exec.Command("cat", "output.txt")

	output, err := cmd.Output()

	if err == nil {
		fmt.Println(string(output))
	}
}
