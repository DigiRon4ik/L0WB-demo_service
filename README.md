<p align="center">
  <img src="https://user-images.githubusercontent.com/25181517/192149581-88194d20-1a37-4be8-8801-5dc0017ffbbe.png" width="100">
</p>
<h1 align="center">L0WB-demo_service</h1>
<h3 align="center">The simplest microservice made for the test task «L0» on the course <a href="https://tech.wildberries.ru/courses/golang/application">«Gorutin Golang»</a> in <a href="https://tech.wildberries.ru/">«TechSchool»</a> from <a href="https://www.wildberries.ru/">«Wildberries»</a></h3>
<p align="center">The service subscribes to a specific topic in Kafka and listens indefinitely. It receives the message as JSON, converts it into a structure and saves it in the database and memcached. You can retrieve saved messages by UID via web-interface.</p>


---

### — _DataBase:_
![PostgreSQL](https://img.shields.io/badge/postgreSQL-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)

### — _Broker:_
![Apache Kafka](https://img.shields.io/badge/Apache%20Kafka-000?style=for-the-badge&logo=apachekafka)

---

### — _How to Install and Use:_
- **Install and Start the Service:**
  ```bash
  git clone https://github.com/DigiRon4ik/L0WB-demo_service.git
  cd L0WB-demo_service
  docker-compose up -d
  ```
- **Send Message to Kafka:**
  ```bash
  cd L0WB-demo_service
  make send
  ```
  **OR**
  ```bash
  cd L0WB-demo_service
  go run cmd/send/main.go
  ```
> [!WARNING]
> Before you can send messages to Kafka, you must have Golang installed on your PC and run the go mod tidy command. The script for sending a message is for demonstration purposes only and is not related to the service. Thank you for your understanding.
- **Data Retrieval:**
  - Use the web interface at [localhost:8080](http://localhost:8080/) to retrieve the data. Or via API _«/order/{uid}»_

---

### Video:
<p align="center">
  <a href="https://youtu.be/5S7tCNepatA">
    <img src="https://i.imgur.com/ma6wGIF.png" >
  </a>
</p>

---

### Screenshot:
<p align="center">
  <img src="https://i.imgur.com/LreGtB1.png" >
</p>

---
