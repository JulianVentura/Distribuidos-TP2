package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
)

func parametrized_work(
	input string,
	permalink_regex *regexp.Regexp,
) (string, error) {

	log.Debugf("Received: %v", input)

	permalink_idx := 7
	body_idx := 8
	sentiment_idx := 9

	splits := strings.Split(input, ",")

	if len(splits) != 11 {
		return "", fmt.Errorf("invalid input")
	}

	permalink := splits[permalink_idx]
	body := splits[body_idx]
	sentiment := splits[sentiment_idx]

	if len(body) < 1 || body == "NaN" {
		return "", fmt.Errorf("invalid input")
	}
	_, err := strconv.ParseFloat(sentiment, 64)
	if err != nil {
		return "", fmt.Errorf("invalid input")
	}
	post_id, err := get_post_id(permalink_regex, permalink)

	if err != nil {
		return "", fmt.Errorf("invalid input")
	}

	output := []string{post_id, sentiment, body}
	result := strings.Join(output, ",")
	return result, nil
}

//TODO: Chequear por valores nulos que puedan tener representación en string, como "NaN" o "nil"
//Lo que puede pasar es que en el body por ejemplo haya algún Null que en pandas es filtrado y acá no
//Idea: Guardar los datasets luego de hacer los filtrados de posts y comments y comparar con la salida de estos filtros
//Habría que usar un único digestor y apagar todos los logs. Luego imprimir por pantalla con fmt las entradas que pasan la validacion

//TODO: Refactor
func generate_regex() (*regexp.Regexp, *regexp.Regexp, *regexp.Regexp) {
	//Post id
	permalink_reg := `https:\/\/old\.reddit\.com\/r\/meirl\/comments\/([^\/]+)\/meirl\/.*`

	//Sentiment
	sentiment_reg := "^-?[0-9]+$" //Support negative

	//Body
	body_reg := `.+` //Non empty char, all chars allowed

	//Build the regex
	perma_r := regexp.MustCompile(permalink_reg)
	body_r := regexp.MustCompile(body_reg)
	sentiment_r := regexp.MustCompile(sentiment_reg)

	return perma_r, body_r, sentiment_r
}

func get_post_id(reg *regexp.Regexp, input string) (string, error) {

	split := reg.FindStringSubmatch(input)
	if len(split) < 2 {
		return "", fmt.Errorf("Could not get id from input")
	}

	return split[1], nil
}

func work_callback() func(string) (string, error) {
	a, _, _ := generate_regex()
	return func(input string) (string, error) { return parametrized_work(input, a) }
}

func test_function() {
	lines := []string{
		//Wront: me_irl instead of meirl
		"12052031,comment,cag9p9z,2vegg,me_irl,False,1370912170,https://old.reddit.com/r/me_irl/comments/1g2h1a/me_irl/cag9p9z/,According to elsewhere on Reddit: Peggle 2,0.0,2",
		//Correct
		"1714671,comment,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.reddit.com/r/meirl/comments/30241o/meirl/cqiw1bf/,Thanks me too,0.4404,1",
		//Fewer values
		"1714671,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.reddit.com/r/meirl/comments/30241o/meirl/cqiw1bf/,Thanks me too,0.4404,1",
		//Bad body
		"1714671,comment,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.reddit.com/r/meirl/comments/30241o/meirl/cqiw1bf/,,0.4404,1",
		//Bad sentiment
		"1714671,comment,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.reddit.com/r/meirl/comments/30241o/meirl/cqiw1bf/,Thanks me too,,1",
		//Bad permalink
		"1714671,comment,cqiw1bf,2s5ti,meirl,False,1429562851,https://old.ret.com/r/meirl/comments/30241o/meirl/cqiw1bf/,Thanks me too,0.4404,1",
	}

	work := work_callback()

	for id, line := range lines {
		result, err := work(line)
		if err != nil {
			fmt.Printf("Line %v: Invalid\n", id+1)
		} else {
			fmt.Printf("Line %v: Valid\n", id+1)
			fmt.Printf("- %v\n", result)
		}
	}
}
