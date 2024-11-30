package main

import "fmt"

type linkRepo struct {
	repo map[string]string
}

func NewLinkRepo() linkRepo {
	return linkRepo{repo: make(map[string]string, 100)}
}

func (r *linkRepo) Create(h string, l string) error {
	r.repo[h] = l
	return nil
}

func (r *linkRepo) Get(h string) (string, error) {

	l, err := r.repo[h]

	if !err {
		return "", fmt.Errorf("CAN'T FIND LINK BY HASH")
	}

	return l, nil
}
