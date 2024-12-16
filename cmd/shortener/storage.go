package main

import "fmt"

const REPO_LENGTH int64 = 100

type linkRepo struct {
	repo map[string]string
}

func NewLinkRepo() linkRepo {
	return linkRepo{repo: make(map[string]string, REPO_LENGTH)}
}

func (r *linkRepo) Create(h, l string) {
	r.repo[h] = l
}

func (r *linkRepo) Get(h string) (string, error) {
	l, err := r.repo[h]

	if !err {
		return "", fmt.Errorf("CAN'T FIND LINK BY HASH")
	}

	return l, nil
}
