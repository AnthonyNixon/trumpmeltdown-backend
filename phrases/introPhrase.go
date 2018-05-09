package phrases

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
)

type JsonFile struct {
	Phrases []Phrase `json:"phrases"`
}

func GetIntroPhrase(meltdownPct int) string {
	// Read the JSON file
	jsonFile, err := os.Open("phrases.json")
	if err != nil {
		fmt.Errorf("%s\n", err)
	}
	defer jsonFile.Close()
	JsonContents, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		fmt.Errorf("%s\n", err)
	}

	fmt.Printf("%d\n", len(JsonContents))
	var jsonStruct JsonFile

	err = json.Unmarshal(JsonContents, &jsonStruct)
	if err != nil {
		fmt.Errorf("%s\n", err)
	}

	// Parse into array of phrases
	phrases := jsonStruct.Phrases

	// Randomly Choose one
	index := rand.Int() % len(phrases)
	phrase := phrases[index]

	// Parse it with a switch statement on the type
	switch phrase.Type {
	case "percentage":
		return fmt.Sprintf(phrase.Format, meltdownPct)
	case "repeat-char-out-of-10":
		charAmount := int(meltdownPct/10) + 1
		charString := ""
		for i := 0; i < charAmount; i++ {
			charString += phrase.Char
		}
		return fmt.Sprintf(phrase.Format, charString)
	case "out-of-5":
		return fmt.Sprintf(phrase.Format, int(meltdownPct/20)+1)
	case "out-of-10":
		return fmt.Sprintf(phrase.Format, int(meltdownPct/10)+1)
	default:
		return fmt.Sprintf("This Tweet is a %d%% meltdown.", meltdownPct)
	}
}
