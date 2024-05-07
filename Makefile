up: interface-build loader-build
	@cd build && docker-compose up -d

down:
	@cd build && docker-compose down --remove-orphans

index:
	@cd build && docker-compose run --rm loading-data ./loading-data -f materials/db.csv

interface-build:
	go build -o simplest-interface cmd/simplestInterface/main.go

loader-build:
	go build -o loading-data cmd/loadingData/main.go

interface-docker-build:
	docker build --file=build/docker/simplestInterface.docker --tag simplest-interface .

interface-run:
	docker run --rm -d -p 8888:8888 --name simplest-interface simplest-interface

loader-docker-build:
	docker build --file=build/docker/loadingData.docker --tag loading-data .

elastic-run:
	docker run --rm -d -p 9200:9200 -p 9300:9300 \
	-e "discovery.type=single-node" \
	-e "xpack.security.enabled=false" \
	--name elasticsearch elasticsearch:8.13.0

clean:
	rm -rf simplest-interface loading-data
