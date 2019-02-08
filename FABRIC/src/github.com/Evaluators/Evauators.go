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

// EvaluatorChaincode example simple Chaincode implementation
type EvaluatorChaincode struct {
}

// ============================================================================================================================
// Structure of assets
// ============================================================================================================================

type TechRepu struct {
	UniqueTechName string `json:"UniqueTechName"`
	AttainedRepu   int    `json:"AttainedRepo"`
	CreatedON      string `json:"createdOn"`
}

// ============================================================================================================================
// Asset Definitions - The ledger will store evaluators and owners
// ============================================================================================================================

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
	err := shim.Start(new(EvaluatorChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode - %s", err)
	}
}

// ============================================================================================================================
// Init - initialize the chaincode
// ============================================================================================================================
func (t *EvaluatorChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Evaluator Store Channel Is Starting Up")
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
func (t *EvaluatorChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println(" ")
	fmt.Println("starting invoke, for - " + function)

	// Handle different functions
	if function == "addAnEvaluator" { //create a new marble
		return addAnEvaluator(stub, args)
	} else if function == "bumpUpEvaluatorRepu" { //create a new marble
		return bumpUpEvaluatorRepu(stub, args)
	} else if function == "getEvaluatorById" {
		return getEvaluatorById(stub, args)
	} else if function == "queryEvaluatorById" {
		return queryEvaluatorById(stub, args)
	} else if function == "updateTheEvaluatedAnswers" {
		return updateTheEvaluatedAnswers(stub, args)
	}

	// error out
	fmt.Println("Received unknown invoke function name - " + function)
	return shim.Error("Received unknown invoke function name - '" + function + "'")
}

// ============================================================================================================================
// Query - legacy function
// ============================================================================================================================
func (t *EvaluatorChaincode) Query(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("Unknown supported call - Query()")
}

func read_everything(stub shim.ChaincodeStubInterface) pb.Response {
	type Everything struct {
		Evaluators []Evaluator `json:"evaluators"`
	}
	var everything Everything

	// ---- Get All Evaluators ---- //
	resultsIterator, err := stub.GetStateByRange("m0", "m9999999999999999999")
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	for resultsIterator.HasNext() {
		aKeyValue, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		queryKeyAsStr := aKeyValue.Key
		queryValAsBytes := aKeyValue.Value
		fmt.Println("on evaluator id - ", queryKeyAsStr)
		var evaluator Evaluator
		json.Unmarshal(queryValAsBytes, &evaluator)                      //un stringify it aka JSON.parse()
		everything.Evaluators = append(everything.Evaluators, evaluator) //add this marble to the list
	}
	fmt.Println("evaluators array - ", everything.Evaluators)

	//change to array of bytes
	everythingAsBytes, _ := json.Marshal(everything) //convert to array of bytes
	return shim.Success(everythingAsBytes)
}

// ============================================================================================================================
// Get history of asset
// ============================================================================================================================
func getHistory(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	type AuditHistory struct {
		TxId  string    `json:"txId"`
		Value Evaluator `json:"value"`
	}
	var history []AuditHistory
	var evaluator Evaluator

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	evaluatorId := args[0]
	fmt.Printf("- start getHistoryForMarble: %s\n", evaluatorId)

	// Get History
	resultsIterator, err := stub.GetHistoryForKey(evaluatorId)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	for resultsIterator.HasNext() {
		historyData, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		var tx AuditHistory
		tx.TxId = historyData.TxId                    //copy transaction id over
		json.Unmarshal(historyData.Value, &evaluator) //un stringify it aka JSON.parse()
		if historyData.Value == nil {                 //marble has been deleted
			var emptyEvaluator Evaluator
			tx.Value = emptyEvaluator //copy nil marble
		} else {
			json.Unmarshal(historyData.Value, &evaluator) //un stringify it aka JSON.parse()
			tx.Value = evaluator                          //copy marble over
		}
		history = append(history, tx) //add this tx to the list
	}
	fmt.Printf("- getHistoryForEvaluator returning:\n%s", history)

	//change to array of bytes
	historyAsBytes, _ := json.Marshal(history) //convert to array of bytes
	return shim.Success(historyAsBytes)
}

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

func addAnEvaluator(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting addAnEvaluator")

	if len(args) != 3 {
		fmt.Println("initEvaluator(): Incorrect number of arguments. Expecting 3 ")
		return shim.Error("intEvaluator(): Incorrect number of arguments. Expecting 3 ")
	}

	//input sanitation
	err1 := sanitize_arguments(args)
	if err1 != nil {
		return shim.Error("Cannot sanitize arguments")
	}

	evaluatorInitialTechName := args[0]
	evaluatorID := args[1]
	fmt.Println(args)
	//check if marble id already exists
	evaluatorAsBytes, err := stub.GetState(evaluatorID)
	if err != nil { //this seems to always succeed, even if key didn't exist
		return shim.Error("error in finding evaluator for - " + evaluatorID)
	}
	if evaluatorAsBytes != nil {
		fmt.Println("This evaluator already exists - " + evaluatorID)
		return shim.Error("This evaluator already exists - " + evaluatorID) //all stop a marble by this id exists
	}

	evaluatorTechRepuObject, err := CreateEvaluatorTechRepuObject(evaluatorInitialTechName)

	evaluatorObject, err := CreateEvaluatorObject(args[1:], evaluatorTechRepuObject)
	if err != nil {
		errorStr := "initEvaluator() : Failed Cannot create object buffer for write : " + args[0]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	fmt.Println(evaluatorObject)
	buff, err := EvaltoJSON(evaluatorObject)
	if err != nil {
		return shim.Error("unable to convert evaluator to json")
	}

	err = stub.PutState(evaluatorID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end addAnEvaluator")
	return shim.Success(nil)
}

// CreateAssetObject creates an asset
func CreateEvaluatorObject(args []string, techRepu TechRepu) (Evaluator, error) {
	var myEvaluator Evaluator

	fmt.Println(args)
	fmt.Println(techRepu)
	// Check there are 10 Arguments provided as per the the struct
	if len(args) != 2 {
		strErr := "CreateEvaluatorObject(): Incorrect number of arguments. Expecting 2 but got " + strconv.Itoa(len(args))
		fmt.Println(strErr)
		return myEvaluator, errors.New(strErr)
	}
	dummyTechRepuArray := []TechRepu{}
	dummyTechRepuArray = append(dummyTechRepuArray, techRepu)

	rawEvalSecret := args[1]

	hashedpassword, err := HashPassword(rawEvalSecret)
	if err != nil {

		return myEvaluator, errors.New("error in hashing the password")
	}
	fmt.Println("hashed password is: " + hashedpassword)

	strArr := []string{}
	myEvaluator = Evaluator{args[0], hashedpassword, strArr, dummyTechRepuArray, time.Now().Format("20060102150405")}
	return myEvaluator, nil
}

// CreateAssetObject creates an asset
func CreateEvaluatorTechRepuObject(techName string) (TechRepu, error) {
	techRepu := TechRepu{techName, 10, time.Now().Format("20060102150405")}
	return techRepu, nil
}

func EvaltoJSON(eval Evaluator) ([]byte, error) {

	djson, err := json.Marshal(eval)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return djson, nil
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

// query callback representing the query of a chaincode
func getEvaluatorById(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("sarting the getEvaluatorById() with the args: ")
	fmt.Println(args)
	fmt.Println("========================")
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	evaluatorID := args[0]

	// Get the state from the ledger
	evaluatorbytes, err := stub.GetState(evaluatorID)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + evaluatorID + "\"}"
		return shim.Error(jsonResp)
	}

	if evaluatorbytes == nil {
		jsonResp := "{\"Error\":\"Nil data for " + evaluatorID + "\"}"
		return shim.Error(jsonResp)
	}

	jsonResp := "{\"EvaluatorID\":\"" + evaluatorID + "\",\"data\":\"" + string(evaluatorbytes) + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return shim.Success(evaluatorbytes)
}

// very important as it is required by the Answer chaincode to query
func queryEvaluatorById(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	evaluatorID := args[0]

	queryString := fmt.Sprintf("{\"selector\":{\"EvaluatorID\":\"%s\"}}", evaluatorID)

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

func registerEvaluator(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting registerEvaluator")

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	//input sanitation
	err = sanitize_arguments(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	evaluatorID := args[0]
	rawEvaluatorSecret := args[1]

	// hash the evaluator secret and replace the original one
	args[1], err = HashPassword(rawEvaluatorSecret)

	evaluatorAsBytes, err := stub.GetState(evaluatorID)
	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in finding Evaluator - " + evaluatorID)
		return shim.Error("error in finding evaluator for - " + evaluatorID)
	}

	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in finding Evaluator - " + evaluatorID)
		return shim.Error("error in finding evaluator for - " + evaluatorID)
	}

	str := fmt.Sprintf("%s", evaluatorAsBytes)
	fmt.Println("string is " + str)

	if evaluatorAsBytes != nil {
		fmt.Println("This evaluator already exists - " + evaluatorID)
		return shim.Error("This evaluator already exists - " + evaluatorID) //all stop a marble by this id exists
	}

	techRepuObject, err := CreateEvaluatorTechRepuObject(args[2])
	if err != nil {
		errorStr := "initStudent() : Failed Cannot create object buffer for write : " + args[0]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	evaluatorObject, err := CreateEvaluatorObject(args[0:], techRepuObject)
	if err != nil {
		errorStr := "initStudent() : Failed Cannot create object buffer for write : " + args[0]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	fmt.Println(evaluatorObject)
	buff, err := EvaltoJSON(evaluatorObject)
	if err != nil {
		return shim.Error("unable to convert evaluator to json")
	}

	err = stub.PutState(evaluatorID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end registerEvaluator")
	return shim.Success(nil)
}

func bumpUpEvaluatorRepu(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting bumpUpEvaluatorRepu")

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	//input sanitation
	err = sanitize_arguments(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	evaluatorID := args[0]
	techName := args[1]
	upCount, _ := strconv.Atoi(args[2])

	fmt.Println("bumping up by: ")
	fmt.Println(upCount)

	evaluatorAsBytes, err := stub.GetState(evaluatorID)
	str := fmt.Sprintf("%s", evaluatorAsBytes)
	fmt.Println("string is " + str)

	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in finding Evaluator - " + evaluatorID)
		return shim.Error("error in finding evaluator for - " + evaluatorID)
	}

	dat, err := JSONtoEval(evaluatorAsBytes)
	if err != nil {
		return shim.Error("unable to convert jsonToDoc for" + evaluatorID)
	}

	flag := false

	for i, techRepuData := range dat.EvaluatorTechRepus {
		if techRepuData.UniqueTechName == techName {
			dat.EvaluatorTechRepus[i].AttainedRepu += upCount
			flag = true
			break
		}
	}

	if !flag {
		return shim.Error("tech repu not found for evaluator " + evaluatorID)
	}

	updatedEvaluator := Evaluator{dat.EvaluatorID, dat.EvaluatorSecret, dat.EvaluatedAnswers, dat.EvaluatorTechRepus, dat.CreatedON}

	buff, err := EvaltoJSON(updatedEvaluator)
	if err != nil {
		errorStr := "updateDispatchOrder() : Failed Cannot create object buffer for write : " + args[1]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	err = stub.PutState(dat.EvaluatorID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end addAnEvaluator")
	return shim.Success(nil)
}

func updateTheEvaluatedAnswers(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting updateTheEvaluatedAnswers")

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	//input sanitation
	err = sanitize_arguments(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	evaluatorID := args[0]
	answerHashID := args[1]

	evaluatorAsBytes, err := stub.GetState(evaluatorID)
	str := fmt.Sprintf("%s", evaluatorAsBytes)
	fmt.Println("string is " + str)

	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in finding Evaluator - " + evaluatorID)
		return shim.Error("error in finding evaluator for - " + evaluatorID)
	}

	dat, err := JSONtoEval(evaluatorAsBytes)
	if err != nil {
		return shim.Error("unable to convert jsonToDoc for" + evaluatorID)
	}
	evalAnswers := dat.EvaluatedAnswers
	if contains(evalAnswers, answerHashID) {
		errorStr := "already evaluated cant evaluate the same answer again "
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}
	evalAnswers = append(evalAnswers, answerHashID)

	updatedEvaluator := Evaluator{dat.EvaluatorID, dat.EvaluatorSecret, evalAnswers, dat.EvaluatorTechRepus, dat.CreatedON}

	buff, err := EvaltoJSON(updatedEvaluator)
	if err != nil {
		errorStr := "updateDispatchOrder() : Failed Cannot create object buffer for write : " + args[1]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	err = stub.PutState(dat.EvaluatorID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end updateTheEvaluatedAnswers")
	return shim.Success(nil)
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

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
