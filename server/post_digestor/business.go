package main

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

func parametrized_work(
	input string,
	post_id_regex *regexp.Regexp,
	meme_url_regex *regexp.Regexp,
	post_score_regex *regexp.Regexp,
) (string, error) {

	log.Debugf("Received: %v", input)

	post_id_idx := 2
	post_meme_idx := 9
	post_score_idx := 12

	splits := strings.Split(input, ",")

	if len(splits) != 13 {
		return "", fmt.Errorf("invalid input")
	}

	post_id := splits[post_id_idx]
	post_meme := splits[post_meme_idx]
	post_score := splits[post_score_idx]

	// ok := true
	// ok = ok && post_id_regex.MatchString(post_id)
	// ok = ok && meme_url_regex.MatchString(post_meme)
	// ok = ok && post_score_regex.MatchString(post_score)
	// if !ok {
	// 	return "", fmt.Errorf("invalid input")
	// }

	output := []string{post_id, post_meme, post_score}

	return strings.Join(output, ","), nil
}

//TODO: Chequear por valores nulos que puedan tener representación en string, como "NaN" o "nil"
func generate_regex() (*regexp.Regexp, *regexp.Regexp, *regexp.Regexp) {
	//Post id
	post_id_reg := "^[aA-zZ0-9]+$"

	//Meme url
	protocol := "https?:\\/\\/(www\\.)?"
	host := "[-a-zA-Z0-9@:%._\\+~#=]{1,256}"
	uri := "([-a-zA-Z0-9()@:%_\\+.~#?&//=]*)"
	format := "\\.(jpg|png|jpeg|gif)"

	meme_url_reg := fmt.Sprintf("%s%s%s%s", protocol, host, uri, format)

	//Score
	post_score_reg := "^-?[0-9]+$" //Support negative

	//Build the regex
	post_id_r := regexp.MustCompile(post_id_reg)
	meme_url_r := regexp.MustCompile(meme_url_reg)
	post_score_r := regexp.MustCompile(post_score_reg)

	return post_id_r, meme_url_r, post_score_r
}

func work_callback() func(string) (string, error) {
	a, b, c := generate_regex()
	return func(input string) (string, error) { return parametrized_work(input, a, b, c) }
}

func test_function() {
	lines := []string{
		//Correct
		"3362746,post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,10",
		//Fewer values
		"3362746,post,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,10",
		//Bad score
		"3362746,post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,word",
		//Empty Score
		"3362746,post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,",
		//Empty id
		"3362746,post,,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,10",
		//Bad id
		"3362746,post,@#$%@,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,http://i.imgur.com/Df6K0.gif,,me irl,10",
		//Bad meme url
		"3362746,post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,bad-format,,me irl,10",
		//Empty meme url
		"3362746,post,1258am,2vegg,me_irl,False,1351287079,https://old.reddit.com/r/me_irl/comments/1258am/me_irl/,i.imgur.com,,,me irl,10",
	}

	work := work_callback()

	for id, line := range lines {
		result, err := work(line)
		if err != nil {
			fmt.Printf("Line %v: Invalid\n", id+1)
		} else {
			fmt.Printf("Line %v: Valid\n", id+1)
			fmt.Printf("%v\n", result)
		}
	}
}
