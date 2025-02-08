# Realtime Chat Backend
[Starter Template](https://github.com/Massad/gin-boilerplate)


# **Test Task: Real-Time Chat Backend**

## **Objective**

Develop a **basic real-time chat backend** using **Go** and **Tinode**. The backend should allow users to register, log in, and exchange messages in a **single chat room** (`general`). MongoDB will be used for user authentication and message storage.

---

## **Features**

### **1️⃣ User Authentication**

- Users can **sign up** with email and password.
- Users can **log in** and receive a JWT authentication token.

### **2️⃣ Real-Time Messaging**

- Users can **send** messages to a shared `general` chat room.
- Users can **receive** real-time messages from Tinode.
- Messages should be **persisted** in MongoDB.

### **3️⃣ Message Retrieval**

- API to **fetch** the last 50 messages from `general`.

### **4️⃣ Documentation & Testing**

- Write a **clear `README.md`** with setup instructions.
- Basic testing with Postman or `cURL`.

---

## **Tech Stack**

- **Go (Gin Framework)** – API development.
- **MongoDB** – User and message storage.
- **Tinode** – Real-time messaging.
- **JWT** – Authentication.

---

## **Deliverables**

- **GitHub Repository** containing:
    - Source code (`main.go`, `handlers.go`, `models.go`).
    - `README.md` with setup & API usage instructions.
  
## **API Endpoints**

### **1️⃣ User Authentication**

### **POST /signup**

- **Request:**
    
    ```json
    { "email": "user@example.com", "password": "password123" }
    
    ```
    
- **Response:**
    
    ```json
    { "message": "User registered successfully" }
    
    ```
    

### **POST /login**

- **Request:**
    
    ```json
    { "email": "user@example.com", "password": "password123" }
    
    ```
    
- **Response:**
    
    ```json
    { "token": "jwt-token" }
    
    ```
    

---

### **2️⃣ Messaging**

### **POST /message** (Authenticated)

- **Request:**
    
    ```json
    { "content": "Hello, world!" }
    
    ```
    
- **Response:**
    
    ```json
    { "message": "Message sent successfully" }
    
    ```
    

### **GET /messages** (Authenticated)

- **Response:**

    ```json
    [
        { "author": "user1", "content": "Hello!", "timestamp": "2024-08-08T12:00:00Z" }
    ]
    ```

## **Evaluation Criteria**

- **Code Quality** – Clean, structured, and documented.
- **Correctness** – Meets requirements and works as expected.
- **Security** – JWT authentication & password hashing.
- **Database Integration** – Users & messages stored properly.
- **Real-time Messaging** – Tinode integration.

---

## **Final Notes**

📌 **Focus on functionality** rather than full feature completeness.

✅ **Bonus (Optional, if time allows)** – Docker setup for easy deployment.

