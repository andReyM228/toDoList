package main

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/gookit/color.v1"
)

type Task struct {
	Id          primitive.ObjectID `bson:"id"`
	CreatedAt   time.Time          `bson:"created_at"`
	Name        string             `bson:"name"`
	userId      primitive.ObjectID `bson:"user_id"`
	Description string             `bson:"description"`
	Completed   bool               `bson:"completed"`
}

type User struct {
	Id       primitive.ObjectID `bson:"id"`
	Name     string             `bson:"name"`
	Password string             `bson:"password"`
}

const mongoDBURI = "mongodb://localhost:27017"

var taskCollection *mongo.Collection
var userCollection *mongo.Collection
var ctx = context.TODO()

func init() {
	clientOptions := options.Client().ApplyURI(mongoDBURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}

	taskCollection = client.Database("todo").Collection("task")
	userCollection = client.Database("todo").Collection("user")
}

func main() {
	app := &cli.App{
		Name:  "todo",
		Usage: "A simple cli programme, to manage your tasks",
		Action: func(*cli.Context) error {
			fmt.Println("boom! I say!")
			return nil
		},
		Commands: []*cli.Command{
			&cli.Command{
				Name:      "add",
				Aliases:   []string{"a"},
				ArgsUsage: "adds your task to the list",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()

					task := &Task{
						Name:        args[0],
						Description: args[1],
						CreatedAt:   time.Now().UTC(),
						Completed:   false,
					}

					return createTask(task)
				},
			},
			&cli.Command{
				Name:      "delete",
				Aliases:   []string{"d"},
				ArgsUsage: "removes your task from the list",
				Action: func(c *cli.Context) error {
					args := c.Args().First()
					err := deleteTask(args)
					if err != nil {
						return err
					}

					return nil
				},
			},
			&cli.Command{
				Name:      "viev",
				ArgsUsage: "viev your tasks in the list",
				Action: func(c *cli.Context) error {
					tasks, err := getAll()
					if err != nil {
						if err == mongo.ErrNoDocuments {
							fmt.Print("Nothing to see here.")
							return nil
						}

						return err
					}

					vievTasks(tasks)
					return nil
				},
			},
			&cli.Command{
				Name:      "compliete",
				Aliases:   []string{"c"},
				ArgsUsage: "compliete your task in the list",
				Action: func(c *cli.Context) error {
					args := c.Args().First()
					return completeTask(args)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func createTask(task *Task) error {
	_, err := taskCollection.InsertOne(ctx, task)
	return err
}

func getAll() ([]*Task, error) {
	// passing bson.D{{}} matches all documents in the collection
	filter := bson.D{{}}
	return filterTasks(filter)
}

func filterTasks(filter interface{}) ([]*Task, error) {
	// A slice of tasks for storing the decoded documents
	var tasks []*Task

	cur, err := taskCollection.Find(ctx, filter)
	if err != nil {
		return tasks, err
	}

	for cur.Next(ctx) {
		var t Task
		err := cur.Decode(&t)
		if err != nil {
			return tasks, err
		}

		tasks = append(tasks, &t)
	}

	if err := cur.Err(); err != nil {
		return tasks, err
	}

	// once exhausted, close the cursor
	cur.Close(ctx)

	if len(tasks) == 0 {
		return tasks, mongo.ErrNoDocuments
	}

	return tasks, nil
}

func deleteTask(text string) error {
	filter := bson.D{primitive.E{Key: "text", Value: text}}

	res, err := taskCollection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	if res.DeletedCount == 0 {
		return errors.New("No tasks were deleted")
	}

	return nil
}

func vievTasks(tasks []*Task) {
	for i, v := range tasks {
		if v.Completed {
			color.Green.Printf("%d: %s\n", i+1, v.Name)
		} else {
			color.Yellow.Printf("%d: %s\n", i+1, v.Name)
		}
	}
}

func completeTask(text string) error {
	filter := bson.D{primitive.E{Key: "text", Value: text}}

	update := bson.D{primitive.E{Key: "$set", Value: bson.D{
		primitive.E{Key: "completed", Value: true},
	}}}

	t := &Task{}
	return taskCollection.FindOneAndUpdate(ctx, filter, update).Decode(t)
}
