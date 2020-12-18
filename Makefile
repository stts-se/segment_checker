.PHONY: all
all: zip

segche_lin:
	GOOS=linux GOARCH=amd64 go build -o segche cmd/app_server/main.go

segche_win:
	GOOS=windows GOARCH=amd64 go build -o segche.exe cmd/app_server/main.go


zip: clean segche_lin segche_win
	mkdir dist

	cd dist && \
	mv ../segche ../segche.exe . && \
	cp -r ../cmd/app_server/static/ . && \
	zip -q -r segche.zip segche segche.exe static/ data/ && \
	rm -rf static && \
	rm -f segche segche.exe

	mv dist/segche.zip .
	rmdir dist


clean:
	rm -f segche segche.exe segche.zip
	rm -rf dist
