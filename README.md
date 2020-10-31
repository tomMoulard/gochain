# Gochain
This app is a simple blockchain in go with a web frontend.

## Usage
```bash
make
```

## Configuration
 - `IP`: Bind IP of the go app
 - `PORT`: Bind PORT
 - `POSTGRES_URL`: postgresql URL
 - `POSTGRES_PASSWORD`: postgresql password

## TODO
 - [ ] Create docker image (`Dockerfile`)
    - [ ] Build go
    - [ ] Use Binary
    - [ ] Expose port
 - [ ] Use the service (`docker-compose.yml`)
    - [ ] Build docker image
    - [ ] Configure go and run the app on port 8000
    - [ ] Create postgresql database and link it
    - [ ] Bonus: Create a swagger ui with swaggerapi
