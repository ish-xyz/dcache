package storage

import "fmt"

type RedisStorage struct{}

func (s *RedisStorage) Write(location string, ops string, data interface{}) error {
	fmt.Println("read from redis storage")
	return nil
}

func (s *RedisStorage) Read(location string, ops string) error {
	fmt.Println("write from redis storage")
	return nil
}
