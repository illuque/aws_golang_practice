Build a Lambda with the following command:

"GOARCH=amd64 GOOS=linux go build <lambda_file>.go ; zip <lambda_name>.zip <lambda_name> ; date"