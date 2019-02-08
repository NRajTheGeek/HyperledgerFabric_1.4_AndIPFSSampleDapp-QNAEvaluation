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

// StudentChaincode class
type StudentChaincode struct {
}

// ============================================================================================================================
// Asset Definitions - The ledger will store students and owners
// ============================================================================================================================

type Student struct {
	StudentID         string     `json:"StudentID"`
	StudentSecret     string     `json:"StudentSecret"`
	StudentTechRepus  []TechRepu `json:"StudentTechRepos"`
	AnsweredQuestions []string   `json:"AnsweredQuestions"`
	CreatedON         string     `json:"createdOn"`
}

type TechRepu struct {
	UniqueTechName string `json:"UniqueTechName"`
	AttainedRepu   int    `json:"AttainedRepo"`
	CreatedON      string `json:"createdOn"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
	err := shim.Start(new(StudentChaincode))
	if err != nil {
		fmt.Printf("Error starting Student chaincode - %s", err)
	}
}

func (t *StudentChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	fmt.Println("Student Chaincode Is Starting Up")
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
func (t *StudentChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println(" ")
	fmt.Println("starting invoke, for - " + function)

	// Handle different functions
	if function == "addAStudent" { //create a new marble
		return addAStudent(stub, args)
	} else if function == "bumpUpStudentRepu" { //create a new marble
		return bumpUpStudentRepu(stub, args)
	} else if function == "queryStudentById" {
		return queryStudentById(stub, args)
	} else if function == "updateAnsweredQuestions" {
		return updateAnsweredQuestions(stub, args)
	}

	// error out
	fmt.Println("Received unknown invoke function name - " + function)
	return shim.Error("Received unknown invoke function name - '" + function + "'")
}

// ============================================================================================================================
// Query - legacy function
// ============================================================================================================================
func (t *StudentChaincode) Query(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Error("Unknown supported call - Query()")
}

func addAStudent(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting addAnStudent")

	if len(args) != 3 {
		fmt.Println("initStudent(): Incorrect number of arguments. Expecting 3 ")
		return shim.Error("intStudent(): Incorrect number of arguments. Expecting 3 ")
	}

	//input sanitation
	err1 := sanitize_arguments(args)
	if err1 != nil {
		return shim.Error("Cannot sanitize arguments")
	}

	studentInitialTechName := args[0]
	studentID := args[1]

	//check if marble id already exists
	studentAsBytes, err := stub.GetState(studentID)
	if err != nil { //this seems to always succeed, even if key didn't exist
		return shim.Error("error in finding student for - " + studentID)
	}
	if studentAsBytes != nil {
		fmt.Println("This student already exists - " + studentID)
		return shim.Error("This student already exists - " + studentID) //all stop a marble by this id exists
	}
	studentTechRepuObject, err := CreateStudentTechRepuObject(studentInitialTechName)

	studentObject, err := CreateStudentObject(args[1:], studentTechRepuObject)
	if err != nil {
		errorStr := "initStudent() : Failed Cannot create object buffer for write : " + args[0]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	fmt.Println(studentObject)
	buff, err := StuToJSON(studentObject)
	if err != nil {
		return shim.Error("unable to convert student to json")
	}

	err = stub.PutState(studentID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end addAnStudent")
	return shim.Success(nil)
}

func bumpUpStudentRepu(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting bumpUpStudentRepu")

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	//input sanitation
	err = sanitize_arguments(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	studentID := args[0]
	techName := args[1]

	studentAsBytes, err := stub.GetState(studentID)
	str := fmt.Sprintf("%s", studentAsBytes)
	fmt.Println("string is " + str)

	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in finding Student - " + studentID)
		return shim.Error("error in finding student for - " + studentID)
	}

	dat, err := JSONtoStu(studentAsBytes)
	if err != nil {
		return shim.Error("unable to convert jsonToDoc for" + studentID)
	}

	flag := false

	for i, techRepuData := range dat.StudentTechRepus {
		if techRepuData.UniqueTechName == techName {
			dat.StudentTechRepus[i].AttainedRepu += 10
			flag = true
			break
		}
	}

	if !flag {
		return shim.Error("tech repu not found for student " + studentID)
	}

	updatedStudent := Student{dat.StudentID, dat.StudentSecret, dat.StudentTechRepus, dat.AnsweredQuestions, dat.CreatedON}

	buff, err := StuToJSON(updatedStudent)
	if err != nil {
		errorStr := "updateDispatchOrder() : Failed Cannot create object buffer for write : " + args[1]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	err = stub.PutState(dat.StudentID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end addAnStudent")
	return shim.Success(nil)
}

// very important as it is required by the Answer chaincode to query
func queryStudentById(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	studentID := args[0]

	queryString := fmt.Sprintf("{\"selector\":{\"StudentID\":\"%s\"}}", studentID)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

// query callback representing the query of a chaincode
func getStudentById(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting name of the person to query")
	}

	studentID := args[0]

	// Get the state from the ledger
	studentbytes, err := stub.GetState(studentID)
	if err != nil {
		jsonResp := "{\"Error\":\"Failed to get state for " + studentID + "\"}"
		return shim.Error(jsonResp)
	}

	if studentbytes == nil {
		jsonResp := "{\"Error\":\"Nil data for " + studentID + "\"}"
		return shim.Error(jsonResp)
	}

	jsonResp := "{\"StudentID\":\"" + studentID + "\",\"data\":\"" + string(studentbytes) + "\"}"
	fmt.Printf("Query Response:%s\n", jsonResp)
	return shim.Success(studentbytes)
}

func updateAnsweredQuestions(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	fmt.Println("starting updateStudentAnsweredQuestions")

	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	//input sanitation
	err = sanitize_arguments(args)
	if err != nil {
		return shim.Error(err.Error())
	}

	studentID := args[0]
	answeredQuestionID := args[1]

	studentAsBytes, err := stub.GetState(studentID)

	if err != nil { //this seems to always succeed, even if key didn't exist
		fmt.Println("Error in finding Student - " + studentID)
		return shim.Error("error in finding student for - " + studentID)
	}

	str := fmt.Sprintf("%s", studentAsBytes)
	fmt.Println("string is " + str)

	dat, err := JSONtoStu(studentAsBytes)
	if err != nil {
		return shim.Error("unable to convert jsonToDoc for" + studentID)
	}
	stuAnsweredQuestions := dat.AnsweredQuestions
	if contains(stuAnsweredQuestions, answeredQuestionID) {
		errorStr := "already answered cant repeat "
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}
	stuAnsweredQuestions = append(stuAnsweredQuestions, answeredQuestionID)

	updatedStudent := Student{dat.StudentID, dat.StudentSecret, dat.StudentTechRepus, stuAnsweredQuestions, dat.CreatedON}

	buff, err := StuToJSON(updatedStudent)
	if err != nil {
		errorStr := "updateDispatchOrder() : Failed Cannot create object buffer for write : " + args[1]
		fmt.Println(errorStr)
		return shim.Error(errorStr)
	}

	err = stub.PutState(studentID, buff) //store marble with id as key
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println("- end updateStudentAnsweredQuestions")
	return shim.Success(nil)
}

// ============================================== Private Library ===========================================================

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

// CreateAssetObject creates an asset
func CreateStudentTechRepuObject(techName string) (TechRepu, error) {
	techRepu := TechRepu{techName, 10, time.Now().Format("20060102150405")}
	return techRepu, nil
}

// CreateAssetObject creates an asset
func CreateStudentObject(args []string, techRepu TechRepu) (Student, error) {
	var myStudent Student

	// Check there are 10 Arguments provided as per the the struct
	if len(args) != 2 {
		fmt.Println("CreateStudentObject(): Incorrect number of arguments. Expecting 2 ")
		return myStudent, errors.New("CreateStudentObject(): Incorrect number of arguments. Expecting 2 ")
	}
	dummyTechRepuArray := []TechRepu{}
	dummyTechRepuArray = append(dummyTechRepuArray, techRepu)

	strArr := []string{}

	rawEvalSecret := args[1]

	hashedpassword, err := HashPassword(rawEvalSecret)
	if err != nil {

		return myStudent, errors.New("error in hashing the password")
	}
	myStudent = Student{args[0], hashedpassword, dummyTechRepuArray, strArr, time.Now().Format("20060102150405")}
	return myStudent, nil
}

func StuToJSON(stu Student) ([]byte, error) {

	fmt.Println("stu before being marshelled")
	fmt.Println(stu)

	djson, err := json.Marshal(stu)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return djson, nil
}

func JSONtoStu(data []byte) (Student, error) {

	stu := Student{}
	err := json.Unmarshal([]byte(data), &stu)
	if err != nil {
		fmt.Println("Unmarshal failed : ", err)
		return stu, err
	}

	return stu, nil
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
