coverage:
	go test ./... -coverprofile cover.out.tmp
	cat cover.out.tmp | grep -v "mock_" > cover.out
	go tool cover -func cover.out