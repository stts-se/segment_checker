.PHONY: all
all: zip

segche_lin:
	GOOS=linux GOARCH=amd64 go build -o segche cmd/serv/main.go
	GOOS=linux GOARCH=amd64 go build -o file_server cmd/file_server/*.go

segche_win:
	GOOS=windows GOARCH=amd64 go build -o segche.exe cmd/serv/main.go
	GOOS=windows GOARCH=amd64 go build -o file_server.exe cmd/file_server/*.go


zip: clean segche_lin segche_win
	mkdir dist

	cd dist && \
	mv ../segche ../segche.exe . && \
	mv ../file_server ../file_server.exe . && \
	cp -r ../cmd/serv/static/ . && \
	zip -q -r segche.zip segche segche.exe static/ data/ && \
	rm -rf static && \
	rm -f segche segche.exe file_server file_server.exe 

	mv dist/segche.zip .
	rmdir dist
	


clean:
	rm -f segche segche.exe file_server file_server.exe segche.zip
	rm -rf dist
