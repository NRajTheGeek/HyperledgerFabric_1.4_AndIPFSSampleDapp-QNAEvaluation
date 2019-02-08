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
	"golang.org/x/crypto/bcrypt"
)

// AnswerChaincode example simple Chaincode implementation
type AnswerChaincode struct {
}

func toChaincodeArgs(args ...string) [][]byte {
	bargs := make([][]byte, len(args))
	for i, arg := range args {
		bargs[i] = []byte(arg)
	}
	return bargs
}

// ============================================================================================================================
// Asset Definitions - The ledger will store answers with hash id and cid
// ============================================================================================================================
type Question struct {
	QuestionHashID            string `json:"QuestionHashID"`
	QuestionCID               string `json:"QuestionCID"`
	QuestionerID              string `json:"QuestionerID"`
	QuestionTech              string `json:"QuestionTech"`
	RequiredEvaluatorThumbsUp int    `json:"RequiredEvaluatorThumbsUp"`
	QuestionedOn              string `json:"QuestionedOn'`
}

type Answer struct {
	AnswerHashID              string   `json:"AnswerHashDigest"`
	AnswerCID                 string   `json:"AnswerCID"`
	AnsweredBy                string   `json:"AnsweredBy"`
	QuestionID                string   `json:"QuestionID"`
	EvaluatedBy               []string `json:"EvaluatedBy"`
	AttainedEvaluatorThumbsUp int      `json:"AttainedEvaluatorThumbsUp"`
	AnsweredOn                string   `json:"AnsweredOn"`
}

type TechRepu struct {
	UniqueTechName string `json:"UniqueTechName"`
	AttainedRepu   int    `json:"AttainedRepo"`
	CreatedON      string `json:"createdOn"`
}

type Evaluator struct {
	EvaluatorID        string     `json:"EvaluatorID"`
	EvaluatorSecret    string     `json:"EvaluatorSecret"`
	EvaluatedAnswers   []string   `json:"EvaluatedAnswers"` // only should container TechIDs so we can perform array ops on it with efficiency
	EvaluatorTechRepus []TechRepu `json:"EvaluatorTechRepos"`
	CreatedON          string     `json:"createdOn"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(AnswerChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode - %s", err)
	}
}

// ============================================================================================================================
// Init - initialize the chaincode
// ============================================================================================================================
func (t *AnswerChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Answer Store Channel Is Starting Up")
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
func (t *AnswerChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println(" ")
	fmt.Println("starting invoke, for - " + function)

	// Handle different functions
	if function == "submitAnswer" { //create a new marble
		return submitAnswer(stub, args)
	} else if function == "thumbsUpToAnswer" { //update_answer
		return thumbsUpToAnswer(stub, args)
	} else if function == "queryAnswersByThumsUpCount" { //queryAnswersByStatus
		return queryAnswersByThumsUpCount(stub, args)
	} else if function == "queryAnswerByAnswerHashId" { //queryAnswerStatusByHash
		return queryAnswerByAnswerHashId(stub, args)
	}

	// error out
	fmt.Println("Received unknown invoke function name - " + function)
	return shim.Error("Received unknown invoke function name - '" + function + "'")
}

// ============================================================================================================================
// Query - legacy function
// ============================================================================================================================
func (t *AnswerChaincode) Query(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("Unknown supported call - Query()")
}

// ============================================================================================================================
// Get Answer - get a answer asset from ledger
// ============================================================================================================================
func getAnswer(stub shim.ChaincodeStubInterface, id string) (Answer, error) {
	ans := Answer{}
	answerAsBytes, err := stub.GetState(id) //getState retreives a key/value from the ledger
	if err == nil {                         //this seems to always succeed, even if key didn't exist
		return ans, errors.New("Failed to find marble - " + id)
	}

	if answerAsBytes == nil { //test if marble is actually here or just nil
		return ans, errors.New("Answer does not exist - " + id)
	}

	err = json.Unmarshal([]byte(answerAsBytes), &ans)
	if err != nil {
		fmt.Println("Unmarshal failed : ", err)
		return ans, errors.New("unable to unmarshall")
	}

	fmt.Println(ans)
	return ans, nil
}

func submitAnswer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting submitAnswer")

	if len(args) != 6 {
		fmt.Println("initAnswer(): Incorrect number of arguments. Expecting 6 ")
		return shim.Error("intAnswer(): Incorrect number of arguments. Expecting 6 ")
	}

	//input sanitation
	err1 := sanitize_arguments(args)
	if err1 != nil {
		return shim.Error("Cannot sanitize arguments")
	}
	questionsChaincode := args[0]
	studentsChaincode := args[1]

	answerHashID := args[2]
	// answerCID := args[3]
	answeredBy := args[4]
	questionID := args[5]
	fmt.Println("========================= recieved args ==========================")
	fmt.Println(args)

	// ==================================== check the valid question ===========================================
	channelId := ""
	chainCodeToCall := questionsChaincode //"questions2"
	functionName := "getQuestionById"
	queryKey := questionID

	queryArgs := toChaincodeArgs(functionName, queryKey)
	response := stub.InvokeChaincode(chainCodeToCall, queryArgs, channelId)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to query chaincode. Got error: %s", err.Error())
		fmt.Printf(errStr)
		return shim.Error("error in finding evaluator for - " + questionID)
	}
	questionBytes := response.Payload

	str := fmt.Sprintf("%s", questionBytes)
	fmt.Println("string is " + str)

	questionData, err := JSONtoQues(questionBytes)
	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in unmarshelling - " + questionID)
		return shim.Error("Error in unmarshelling - " + questionID)
	}
	fmt.Println("captured questions data ")
	fmt.Println(questionData)
	// ============================================================================================

	//check if answer id already exists
	answerAsBytes, err := stub.GetState(answerHashID)
	if err != nil { //this seems to always succeed, even if key didn't exist
		return shim.Error("error in finding asnswer for - " + answerHashID)
	}
	if answerAsBytes != nil {
		fmt.Println("This answer already exists - " + answerHashID)
		return shim.Error("This answer already exists - " + answerHashID) //all stop a marble by this id exists
	}

	answerObject, err := CreateAnswerObject(args[2:])
	if err != nil {
		errorStr := "submitAnswer() : Failed Cannot create object buffer for write : " + args[0]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	fmt.Println(answerObject)
	buff, err := AnsToJSON(answerObject)
	if err != nil {
		return shim.Error("unable to convert answer object to json")
	}

	// also update the student ledger for this answer to the question in the student's aswers array

	f := "updateAnsweredQuestions"
	channelID := ""
	chainCodeToCall = studentsChaincode //"students3"

	invokeArgs := toChaincodeArgs(f, answeredBy, questionID)

	response = stub.InvokeChaincode(chainCodeToCall, invokeArgs, channelID)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", string(response.Payload))
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}
	//======================================================================================================

	err = stub.PutState(answerHashID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end submitAnswer")
	return shim.Success(nil)
}

func queryAnswersByThumsUpCount(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	thumbsUpCount := args[0]

	queryString := fmt.Sprintf("{\"selector\":{\"AttainedEvaluatorThumbsUp\":\"%s\"}}", thumbsUpCount)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func queryAnswerByAnswerHashId(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	//   0
	// "bob"
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	answerHashID := args[0]
	queryString := fmt.Sprintf("{\"selector\":{\"AnswerHashID\":\"%s\"}}", answerHashID)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
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

func queryOtherChaincodeByKeyOnly(stub shim.ChaincodeStubInterface, args []string) (pb.Response, error) {

	fmt.Println("starting thumbsUpToAnswer")
	// var arr []byte

	if len(args) != 4 {
		return shim.Error(""), errors.New("Incorrect number of arguments. Expecting 4")
	}

	//input sanitation
	err := sanitize_arguments(args)
	if err != nil {
		return shim.Error(""), errors.New("error in sanitization")
	}
	channelID := args[0]
	chainCodeToCall := args[1]
	functionName := args[2]
	queryKey := args[3]

	queryArgs := toChaincodeArgs(functionName, queryKey)
	response := stub.InvokeChaincode(chainCodeToCall, queryArgs, channelID)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to query chaincode. Got error: %s", err.Error())
		fmt.Printf(errStr)
		return shim.Error(""), errors.New(errStr)
	}
	bytesResponse := response.Payload

	return shim.Success(bytesResponse), nil
}

func getAnswerLedgerState(stub shim.ChaincodeStubInterface, args []string) (Answer, error) {
	// action can be performed
	var myAnswer Answer
	fmt.Println("starting getAnswerLedgerState")

	if len(args) != 1 {
		return myAnswer, errors.New("Incorrect number of arguments. Expecting 1")
	}

	//input sanitation
	err := sanitize_arguments(args)
	if err != nil {
		return myAnswer, errors.New("problem in sanitization")
	}
	answerHashID := args[0]

	answerDataBytes, err := stub.GetState(answerHashID)
	str := fmt.Sprintf("%s", answerDataBytes)
	fmt.Println("string is " + str)

	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in finding Answer  for - " + answerHashID)
		return myAnswer, errors.New("Error in finding Answer  for - " + answerHashID)
	}

	if answerDataBytes == nil {
		jsonResp := "{\"Error\":\"Nil value for " + answerHashID + "\"}"
		return myAnswer, errors.New(jsonResp)
	}
	dat, err := JSONtoAns(answerDataBytes)
	if err != nil {
		return myAnswer, errors.New("unable to convert JSONtoAns for" + answerHashID)
	}

	return dat, nil
}

// for thumbsup first validate the registered evaluator by evaluator secret from the evaluator chaincode
// then allow the evaluator to do a thumsup against an answer hash id
// iff the evaluator has a tech reputation more than 1000
func thumbsUpToAnswer(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	// var jsonResp string

	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 5")
	}

	//input sanitation
	err = sanitize_arguments(args)
	if err != nil {
		return shim.Error(err.Error())
	}
	questionsChaincode := args[0]
	evaluatorsChaincode := args[1]

	answerHashID := args[2]
	evaluatorID := args[3]
	rawEvaluatorSecret := args[4]

	// ================================== Query the question ledger ================================================
	var ledgerQueryList []string
	ledgerQueryList = append(ledgerQueryList, answerHashID)

	dat, err := getAnswerLedgerState(stub, ledgerQueryList)
	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in finding Answer  for - " + answerHashID)
		return shim.Error("error in finding answer for - " + answerHashID)
	}

	channelId := ""
	chainCodeToCall := questionsChaincode //"questions2"
	functionName := "getQuestionById"
	queryKey := dat.QuestionID

	queryArgs := toChaincodeArgs(functionName, queryKey)
	response := stub.InvokeChaincode(chainCodeToCall, queryArgs, channelId)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to query chaincode. Got error: %s", err.Error())
		fmt.Printf(errStr)
		return shim.Error("error in finding evaluator for - " + evaluatorID)
	}
	questionBytes := response.Payload

	str := fmt.Sprintf("%s", questionBytes)
	fmt.Println("string is " + str)

	questionData, err := JSONtoQues(questionBytes)
	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in unmarshelling - " + evaluatorID)
		return shim.Error("Error in unmarshelling - " + evaluatorID)
	}
	answerTech := questionData.QuestionTech

	//  first check that whether this evaluator id and the secret are right from the evaluator chaincode
	//  then check the tech repu of the evaluator
	//  grab evaluator tech repu array by evaluator id and grab evaluator's tech repu as per the tech
	//===============================================================================================

	channelId = ""
	chainCodeToCall = evaluatorsChaincode //"evaluators9"
	functionName = "getEvaluatorById"
	queryKey = evaluatorID

	// evaluatorsData := evaluatorsBytes
	fmt.Println("=======================================================")
	fmt.Println(" =========================== " + queryKey)

	queryArgs = toChaincodeArgs(functionName, queryKey)
	fmt.Println(chainCodeToCall)
	fmt.Println(queryArgs)
	fmt.Println(channelId)
	fmt.Println("=======================================================")

	response = stub.InvokeChaincode(chainCodeToCall, queryArgs, channelId)
	if response.Status != shim.OK {
		errStr := fmt.Sprintf("Failed to query chaincode. Got error: %s", err.Error())
		fmt.Printf(errStr)
		return shim.Error("error in finding evaluator for - " + evaluatorID)
	}
	evaluatorsBytes := response.Payload
	fmt.Println("=======================================================")

	str = fmt.Sprintf("%s", evaluatorsBytes)
	fmt.Println("string is " + str)

	evaluatorsData, err := JSONtoEval(evaluatorsBytes)
	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in unmarshelling - " + evaluatorID)
		return shim.Error("Error in unmarshelling - " + evaluatorID)
	}
	// answerTech = evaluatorsData.QuestionTech

	// now grab and test the evaluator secret if it is right
	hashedEvalSecret := evaluatorsData.EvaluatorSecret

	isSuccess := CheckPasswordHash(rawEvaluatorSecret, hashedEvalSecret)
	if !isSuccess {
		errStr := fmt.Sprintf("not authorized to perform this action. ")
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}
	techRepuArray := evaluatorsData.EvaluatorTechRepus

	flag := false
	attainedTechRepu := 0
	for _, techRepuData := range techRepuArray {
		if answerTech == techRepuData.UniqueTechName {
			flag = true
			attainedTechRepu = techRepuData.AttainedRepu
			break
		}
	}
	if flag && attainedTechRepu > 1000 {

		// First just update the evaluated answers of the evaluator
		f := "updateTheEvaluatedAnswers"
		channelID := ""
		chainCodeToCall = evaluatorsChaincode //"evaluators9"

		invokeArgs := toChaincodeArgs(f, evaluatorID, answerHashID)

		response := stub.InvokeChaincode(chainCodeToCall, invokeArgs, channelID)
		if response.Status != shim.OK {
			errStr := fmt.Sprintf("Failed to invoke chaincode. Got error: %s", string(response.Payload))
			fmt.Printf(errStr)
			return shim.Error(errStr)
		}
		//==========================================================
		evalyBy := dat.EvaluatedBy
		evalyBy = append(evalyBy, evaluatorID)

		thumbsUp := dat.AttainedEvaluatorThumbsUp + 1

		updatedAnswer := Answer{dat.AnswerHashID, dat.AnswerCID, dat.AnsweredBy, dat.QuestionID, evalyBy, thumbsUp, dat.AnsweredOn}

		buff, err := AnsToJSON(updatedAnswer)
		if err != nil {
			errorStr := "updateDispatchOrder() : Failed Cannot create object buffer for write : " + args[1]
			fmt.Println(errorStr)
			return shim.Error(errorStr)
		}

		err = stub.PutState(answerHashID, buff) //store marble with id as key
		if err != nil {
			return shim.Error(err.Error())
		}

		fmt.Println("- end thumbsUpToAnswer")
		return shim.Success(nil)

	} else {
		errStr := fmt.Sprintf("either you dont have required tech repu or the tech repu is less than 1000. ")
		fmt.Printf(errStr)
		return shim.Error(errStr)
	}
}

// ====================================================== Private Library ====================================================

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

// CreateAssetObject creates an asset
func CreateAnswerObject(args []string) (Answer, error) {
	var myAnswer Answer

	strArr := []string{}
	// Check there are 10 Arguments provided as per the the struct
	if len(args) != 4 {
		fmt.Println("CreateAnswerObject(): Incorrect number of arguments. Expecting 4")
		return myAnswer, errors.New("CreateAnswerObject(): Incorrect number of arguments. Expecting 4")
	}

	myAnswer = Answer{args[0], args[1], args[2], args[3], strArr, 0, time.Now().Format("20060102150405")}
	return myAnswer, nil
}

func AnsToJSON(ans Answer) ([]byte, error) {

	djson, err := json.Marshal(ans)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return djson, nil
}

func EvalToJSON(eval Evaluator) ([]byte, error) {

	djson, err := json.Marshal(eval)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return djson, nil
}

func JSONtoAns(data []byte) (Answer, error) {

	ans := Answer{}
	err := json.Unmarshal([]byte(data), &ans)
	if err != nil {
		fmt.Println("Unmarshal failed : ", err)
		return ans, err
	}

	return ans, nil
}

func JSONtoEval(data []byte) (Evaluator, error) {

	eval := Evaluator{}
	err := json.Unmarshal([]byte(data), &eval)
	if err != nil {
		fmt.Println("Unmarshal failed : ", err)
		return eval, err
	}

	return eval, nil
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

func contains(techRepuArray []string, match string) bool {
	flag := false
	for _, data := range techRepuArray {
		if data == match {
			flag = true
			break
		}
	}
	return flag
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
