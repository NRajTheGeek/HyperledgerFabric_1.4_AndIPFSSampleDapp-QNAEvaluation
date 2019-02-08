package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// QuestionChaincode example simple Chaincode implementation
type QuestionChaincode struct {
}

// ============================================================================================================================
// Asset Definitions - The ledger will store questions with hash id and cid
// ============================================================================================================================

type Question struct {
	QuestionHashID            string `json:"QuestionHashID"`
	QuestionCID               string `json:"QuestionCID"`
	QuestionerID              string `json:"QuestionerID"`
	QuestionTech              string `json:"QuestionTech"`
	RequiredEvaluatorThumbsUp int    `json:"RequiredEvaluatorThumbsUp"`
	QuestionedOn              string `json:"QuestionedOn'`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(QuestionChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode - %s", err)
	}
}

// ============================================================================================================================
// Init - initialize the chaincode
// ============================================================================================================================
func (t *QuestionChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Question Store Channel Is Starting Up")
	funcName, args := stub.GetFunctionAndParameters()
	var err error
	txId := stub.GetTxID()

	fmt.Println("  Init() is running")
	fmt.Println("  Transaction ID: ", txId)
	fmt.Println("  GetFunctionAndParameters() function: ", funcName)
	fmt.Println("  GetFunctionAndParameters() args count: ", len(args))
	fmt.Println("  GetFunctionAndParameters() args found: ", args)

	// expecting 1 arg for instantiate or upgrade
	if len(args) == 2 {
		fmt.Println("  GetFunctionAndParameters() : Number of arguments", len(args))
	}
	// this is a very simple test. let's write to the ledger and error out on any errors
	// it's handy to read this right away to verify network is healthy if it wrote the correct value
	err = stub.PutState(args[0], []byte(args[1]))
	if err != nil {
		return shim.Error(err.Error()) //self-test fail
	}

	fmt.Println("Ready for action") //self-test pass
	return shim.Success(nil)
}

// ============================================================================================================================
// Invoke - Our entry point for Invocations
// ============================================================================================================================
func (t *QuestionChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println(" ")
	fmt.Println("starting invoke, for - " + function)

	// Handle different functions
	if function == "submitQuestion" { //create a new marble
		return submitQuestion(stub, args)
	} else if function == "queryQuestionById" {
		return queryQuestionById(stub, args)
	} else if function == "getQuestionById" {
		return getQuestionById(stub, args)
	}

	// error out
	fmt.Println("Received unknown invoke function name - " + function)
	return shim.Error("Received unknown invoke function name - '" + function + "'")
}

// ============================================================================================================================
// Query - legacy function
// ============================================================================================================================
func (t *QuestionChaincode) Query(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("Unknown supported call - Query()")
}

// ============================================================================================================================
// Get Question - get a question asset from ledger
// ============================================================================================================================
func get_question(stub shim.ChaincodeStubInterface, id string) (Question, error) {
	ques := Question{}
	questionAsBytes, err := stub.GetState(id) //getState retreives a key/value from the ledger
	if err == nil {                           //this seems to always succeed, even if key didn't exist
		return ques, errors.New("Failed to find marble - " + id)
	}

	/*fmt.Println("question id from question is " + question.QuestionID)
	fmt.Println("question id of requested question is " + id)*/

	if questionAsBytes == nil { //test if marble is actually here or just nil
		return ques, errors.New("Question does not exist - " + id)
	}

	err = json.Unmarshal([]byte(questionAsBytes), &ques)
	if err != nil {
		fmt.Println("Unmarshal failed : ", err)
		return ques, errors.New("unable to unmarshall")
	}

	fmt.Println(ques)
	return ques, nil
}

func submitQuestion(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting submitQuestion")

	if len(args) != 5 {
		fmt.Println("initQuestion(): Incorrect number of arguments. Expecting 5 ")
		return shim.Error("intQuestion(): Incorrect number of arguments. Expecting 5 ")
	}

	//input sanitation
	err1 := sanitize_arguments(args)
	if err1 != nil {
		return shim.Error("Cannot sanitize arguments")
	}

	questionHashID := args[0]

	//check if marble id already exists
	questionAsBytes, err := stub.GetState(questionHashID)
	if err != nil { //this seems to always succeed, even if key didn't exist
		return shim.Error("error in finding question for - " + questionHashID)
	}
	if questionAsBytes != nil {
		fmt.Println("This question already exists - " + questionHashID)
		return shim.Error("This question already exists - " + questionHashID) //all stop a marble by this id exists
	}

	questionObject, err := CreateQuestionObject(args[0:])
	if err != nil {
		errorStr := "initQuestion() : Failed Cannot create object buffer for write : " + args[0]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	fmt.Println(questionObject)
	buff, err := QuestoJSON(questionObject)
	if err != nil {
		return shim.Error("unable to convert question to json")
	}

	err = stub.PutState(questionHashID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end submitQuestion")
	return shim.Success(nil)
}

// query callback representing the query of a chaincode
func getQuestionById(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	questionID := args[0]

	// Get the state from the ledger
	questionbytes, err := stub.GetState(questionID)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + questionID + "\"}"
		return shim.Error(jsonResp)
	}

	if questionbytes == nil {
		jsonResp := "{\"Error\":\"Nil data for " + questionID + "\"}"
		return shim.Error(jsonResp)
	}

	jsonResp := "{\"QuestionID\":\"" + questionID + "\",\"data\":\"" + string(questionbytes) + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return shim.Success(questionbytes)
}

func queryQuestionById(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "bob"
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	questionHashID := args[0]
	queryString := fmt.Sprintf("{\"selector\":{\"QuestionHashID\":\"%s\"}}", questionHashID)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// =========================================== Private Libraries ========================================================

// ========================================================
// Input Sanitation - dumb input checking, look for empty strings
// ========================================================
func sanitize_arguments(strs []string) error {
	for i, val := range strs {
		if len(val) <= 0 {
			return errors.New("Argument " + strconv.Itoa(i) + " must be a non-empty string")
		}
		// if len(val) > 32 {
		// 	return errors.New("Argument " + strconv.Itoa(i) + " must be <= 32 characters")
		// }
	}
	return nil
}

func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

// CreateAssetObject creates an asset
func CreateQuestionObject(args []string) (Question, error) {
	var myQuestion Question

	// Check there are 10 Arguments provided as per the the struct
	if len(args) != 5 {
		fmt.Println("CreateQuestionObject(): Incorrect number of arguments. Expecting 5 ")
		return myQuestion, errors.New("CreateQuestionObject(): Incorrect number of arguments. Expecting 5 ")
	}
	requiredEvaluatorThumbsUp, _ := strconv.Atoi(args[4])
	myQuestion = Question{args[0], args[1], args[2], args[3], requiredEvaluatorThumbsUp, time.Now().Format("20060102150405")}
	return myQuestion, nil
}

func QuestoJSON(ques Question) ([]byte, error) {

	fmt.Println("ques before being marshelled")
	fmt.Println(ques)

	djson, err := json.Marshal(ques)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return djson, nil
}

func JSONtoQues(data []byte) (Question, error) {

	ques := Question{}
	err := json.Unmarshal([]byte(data), &ques)
	if err != nil {
		fmt.Println("Unmarshal failed : ", err)
		return ques, err
	}

	return ques, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
