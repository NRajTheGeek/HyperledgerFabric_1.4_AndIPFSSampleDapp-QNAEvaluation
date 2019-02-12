# KYC App Using the Hyperledger Fabric 1.4 and IPFS private cloud cluster

* _Pre-requisites:_
  * Docker version 18.09.0+
  * docker-compose version 1.18+ 
  * Node.js v8.5+
  * fabric-client (fabric-sdk) v1.4+
  * fabric-ca-client (fabric-sdk) v1.4+
  * Required docker images declared in the docker-compose files
  * ipfs-http-client (ipfs http library) v29.0+
  * IDE of Your choice (preferred is VsCode)

### Index
1. Introduction
2. The Flow & Idea
3. Chaincode Architecture
4. Chaincode limitations & assumptions
5. Introduction to our Fabric network architecture
6. The IPFS cluster architecture
7. IPFS: our data store (privatized)
8. Steps to create and maintain fabric network before deploying our chaincodes
9. Steps to start the server
10. Swagger for interaction with the server
11. Conclusion and what next?


## 1. Introduction:- 
This is a POC project for demonstarating the inter-communicating go chaincodes of the hyperledger fabric. This also utilizes IPFS cluster as the privatized decentralized data storage medium. 
Here you will find a Fabric Network cluster and an IPFS cluster working to accomplish the demonstartion along with a NodeJS server and a file based record keeping. There are four go chaincodes to maintain ledgers for:

  * Students
  * Evaluators
  * Questions, and
  * Answers

Let's explore the entire idea in the next section.


## 2. The Flow & Idea


## 3. Chaincode Architecture

## 4. Chaincode limitations & assumptions

## 5. Introduction to our Fabric network architecture

## 6. The IPFS cluster architecture

## 7. IPFS: our data store (privatized)


## 8. Steps to create and maintain fabric network before deploying our chaincodes

Image below :-

![]()


## 9. Steps to start the server

From Project root directory
```
npm start 
```
Now the server has started, aps can be accessed at localhost on port 3000.

## 10. Swagger for interaction with the server

To interact with the APIs there is a Swagger UI hosted which dobles as a clean documentation for this server's APIs.

Go to the following link at your browser.


## 11. Conclusion


##  What's next?
This is just to demo the concept of the private transactions over the quorum blockchain network and it is promising in this regard.
We obviously need to improve this alot if we want to make an awesome demo app but even in the current stag it proves the concept.

If You have Ideas of improvement please email me at:

rajputneerajkumar815@gmail.com
