package db

import (
	"strconv"

	"github.com/jsphweid/harmondex/model"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

func GetMidiMetadatas(filenames []string) map[string]model.MidiMetadata {
	if len(filenames) > 10 {
		panic("Not supposed to pass in more than 10 filenames!")
	}

	res := make(map[string]model.MidiMetadata)

	if len(filenames) == 0 {
		return res
	}

	var keys []map[string]*dynamodb.AttributeValue
	for _, filename := range filenames {
		key := make(map[string]*dynamodb.AttributeValue)
		key["PK"] = &dynamodb.AttributeValue{
			S: aws.String(filename),
		}
		keys = append(keys, key)
	}

	endpoint := "http://localhost:8000"
	session, err := session.NewSession(&aws.Config{
		Region:   aws.String("localhost"),
		Endpoint: &endpoint,
	})
	if err != nil {
		panic("Could not create a new DynamoDB session because " + err.Error())
	}

	client := dynamodb.New(session)
	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			"harmondex-metadata": {Keys: keys},
		},
	}
	dbres, err := client.BatchGetItem(input)
	if err != nil {
		panic("Error from DynamoDB: " + err.Error())
	}

	for _, v := range dbres.Responses["harmondex-metadata"] {
		var s model.MidiMetadata
		if v["Year"].N != nil {
			year, _ := strconv.ParseUint(*v["Year"].N, 10, 32)
			s.Year = uint(year)
		}
		s.Artist = *v["Artist"].S
		s.Release = *v["Release"].S
		s.Title = *v["Title"].S
		res[*v["PK"].S] = s
	}

	return res
}
