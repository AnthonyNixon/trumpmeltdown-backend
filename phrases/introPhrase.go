package phrases

import (
	"fmt"
	"math/rand"
)

const NUMINTROS = 6

func GetIntroPhrase(meltdownPct int) string {
	switch rand.Int() % NUMINTROS {
	case 0:
		return fmt.Sprintf("This Tweet is a %d%% meltdown. ", meltdownPct)
	case 1:
		flameAmount := int(meltdownPct / 10)
		flameString := ""
		for i := 0; i < flameAmount; i++ {
			flameString += "🔥"
		}
		return fmt.Sprintf("This Tweet scores %s out of 10. ", flameString)
	case 2:
		return fmt.Sprintf("%d out of 5 dentists think this tweet is a meltdown. ", int(meltdownPct/20)+1)
	case 3:
		return fmt.Sprintf("%d out of 10 trump supporters think this tweet is a meltdown. ", int(meltdownPct/10)+1)
	case 4:
		return fmt.Sprintf("If melting down causes small hands, this tweet means Trump's hands are %d%% smaller. ", meltdownPct)
	case 5:
		return fmt.Sprintf("☝️This Tweet: %d%%☝️", meltdownPct)
	}
	return fmt.Sprintf("This Tweet is a %d%% meltdown. ", meltdownPct)
}
