package main

import "fmt"

const repoLength int64 = 100

type linkRepo struct {
	repo map[string]string
}

func NewLinkRepo() linkRepo {
	return linkRepo{repo: make(map[string]string, repoLength)}
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
