package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoURI = "mongodb://localhost:27017"
	dbName   = "filedb"
	collName = "filecontents"
	numFiles = 100
)

var collection *mongo.Collection

type WriteRequest struct {
	FileID  int    `json:"file_id"`
	Content string `json:"content"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database(dbName).Collection(collName)

	// Initialize files in MongoDB
	initializeFiles(ctx)

	// Setup routes
	r := gin.Default()

	r.POST("/write", writeHandler)
	r.GET("/read/:file_id", readHandler)

	fmt.Println("🚀 Server running at http://localhost:8080")
	r.Run(":8080")
}

func initializeFiles(ctx context.Context) {
	existingCursor, err := collection.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"file_id": 1}))
	if err != nil {
		log.Fatal(err)
	}
	defer existingCursor.Close(ctx)

	existing := make(map[int]bool)
	for existingCursor.Next(ctx) {
		var doc struct {
			FileID int `bson:"file_id"`
		}
		if err := existingCursor.Decode(&doc); err == nil {
			existing[doc.FileID] = true
		}
	}

	var toInsert []interface{}
	for i := 0; i < numFiles; i++ {
		if !existing[i] {
			toInsert = append(toInsert, bson.M{"file_id": i, "lines": []string{}})
		}
	}

	if len(toInsert) > 0 {
		_, err := collection.InsertMany(ctx, toInsert)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("✅ Inserted %d new MongoDB file docs.\n", len(toInsert))
	} else {
		fmt.Println("✅ All 100 MongoDB file docs already exist.")
	}
}

func writeHandler(c *gin.Context) {
	var req WriteRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.FileID < 0 || req.FileID >= numFiles {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file_id"})
		return
	}

	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"file_id": req.FileID},
		bson.M{"$push": bson.M{"lines": req.Content}},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write to MongoDB"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "file_id": req.FileID})
}

func readHandler(c *gin.Context) {
	idStr := c.Param("file_id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 0 || id >= numFiles {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file_id"})
		return
	}

	var result struct {
		FileID int      `bson:"file_id"`
		Lines  []string `bson:"lines"`
	}

	err = collection.FindOne(context.Background(), bson.M{"file_id": id}).Decode(&result)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"file_id": result.FileID, "lines": result.Lines})
}
