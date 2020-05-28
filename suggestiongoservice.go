package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"suggestiongoservice/suggestpb"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var collection *mongo.Collection

type server struct {
}

type Suggestion struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	Email  string             `bson:"email"`
	Detail string             `bson:"detail"`
	Date   string             `bson:"date"`
}

func main() {

	// if we crash the go code, we get the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("Connecting to MongoDB")
	// connect to MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb+srv://sathees:XO0SCPyQnmvEjKFo@cluster0-x3xzb.gcp.mongodb.net/test?retryWrites=true&w=majority"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	collection = client.Database("sugggestdb").Collection("usrsuggest")

	// [START setting_port]
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}

	tls := false
	if tls {
		certFile := "certs/server.crt"
		keyFile := "certs/server.key"
		creds, sslErr := credentials.NewServerTLSFromFile(certFile, keyFile)
		if sslErr != nil {
			log.Fatalf("Failed loading certificates: %v", sslErr)
			return
		}
		opts = append(opts, grpc.Creds(creds))
	}

	s := grpc.NewServer(opts...)
	suggestpb.RegisterSuggestServiceServer(s, &server{})
	//Register reflection service on gRPC server.
	reflection.Register(s)

	go func() {
		fmt.Println("Starting Server...")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Wait for Control C to exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// Block until a signal is received
	<-ch
	// First we close the connection with MongoDB:
	fmt.Println("Closing MongoDB Connection")
	// client.Disconnect(context.TODO())
	if err := client.Disconnect(context.TODO()); err != nil {
		log.Fatalf("Error on disconnection with MongoDB : %v", err)
	}
	// Second step : closing the listener
	fmt.Println("Closing the listener")
	if err := lis.Close(); err != nil {
		log.Fatalf("Error on closing the listener : %v", err)
	}
	// Finally, we stop the server
	fmt.Println("Stopping the server")
	s.Stop()
	lis.Close()
	fmt.Println("End of Program")

}

func (*server) CreateSuggest(ctx context.Context, req *suggestpb.CreateSuggestRequest) (*suggestpb.CreateSuggestResponse, error) {
	fmt.Println("Create blog request")

	suggestData := req.GetSuggest()

	data := Suggestion{
		Email:  suggestData.GetEmail(),
		Detail: suggestData.GetDetail(),
		Date:   suggestData.GetDate(),
	}

	fmt.Println("data:", data)

	res, err := collection.InsertOne(context.Background(), data)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot convert to OID"),
		)
	}

	return &suggestpb.CreateSuggestResponse{
		Suggest: &suggestpb.Suggest{
			Id:     oid.Hex(),
			Email:  suggestData.GetEmail(),
			Detail: suggestData.GetDetail(),
			Date:   suggestData.GetDate(),
		},
	}, nil
}

func (*server) ListSuggest(req *suggestpb.ListSuggestRequest, stream suggestpb.SuggestService_ListSuggestServer) error {
	fmt.Println("List blog request")

	cur, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		data := &Suggestion{}
		err := cur.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
			)

		}
		stream.Send(&suggestpb.ListSuggestResponse{Suggest: dataToSuggestPb(data)})
	}

	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	return nil
}

func dataToSuggestPb(data *Suggestion) *suggestpb.Suggest {
	return &suggestpb.Suggest{
		Id:     data.ID.Hex(),
		Email:  data.Email,
		Detail: data.Detail,
		Date:   data.Date,
	}
}

func (*server) DeleteSuggest(ctx context.Context, req *suggestpb.DeleteSuggestRequest) (*suggestpb.DeleteSuggestResponse, error) {

	fmt.Println("Delete Suggest Request")

	oid, err := primitive.ObjectIDFromHex(req.GetSuggestId())

	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	filter := bson.M{"_id": oid}

	res, err := collection.DeleteOne(context.Background(), filter)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot Delete object: %v", err),
		)
	}

	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find: %v", err),
		)
	}

	return &suggestpb.DeleteSuggestResponse{SuggestId: req.GetSuggestId()}, nil
}
