package main

import (
	"context"
	"fmt"
	"html"

	"github.com/go-git/go-git/v5/plumbing/object"
)

func renderHome(username string, repositories []string) string {
	ret := fmt.Sprintf(`<html>
		<h4>
			<a href="/">home</a>
		</h4>
		<p>Git mirror of %s</p>
		<details open>
			<summary>Projects</summary>
			<ul>`, html.EscapeString(username))

	for _, repository := range repositories {
		ret += fmt.Sprintf(`<li><a href="/%s/tree/">%s</a></li>`, repository, repository)
	}

	return ret + "</details>"
}

func renderContext(repository string, filePath string, branchShort string, branchList []string, fileList []gitFile, commitList []gitCommit) string {
	ret := fmt.Sprintf(`<html>
		<h4>
			<a href="/">home</a> -> <a href="/%s/tree/?branch=%s">%s</a> (%s)
		</h4>`, repository, branchShort, repository, branchShort)

	// Branches
	ret += `<details>
		<summary>Branches</summary>
    	<ul>`
	for _, otherBranchShort := range branchList {
		if otherBranchShort == branchShort {
			ret += fmt.Sprintf("<li>%s</li>", branchShort)
		} else {
			ret += fmt.Sprintf("<li><a href=\"/%s/tree/\">%s</a></li>", repository, otherBranchShort)
		}
	}
	ret += "</ul></details>"

	// Commits
	ret += `<details>
		<summary>Commits</summary>
		<ul>`
	for _, c := range commitList {
		ret += fmt.Sprintf("<li><a href=\"/%s/commit/%s\">%s</a></li>", repository, c.Hash.String(), html.EscapeString(c.Message))
	}
	ret += "</ul></details>"

	// Files
	ret += `<details open>
		<summary>Files</summary>
		<ul>`
	for _, gf := range fileList {
		var link string
		var text string
		if gf.IsDir {
			link = fmt.Sprintf("%s/tree/%s/%s/?branch=%s", repository, filePath, gf.Name, branchShort)
			text = fmt.Sprintf("<li><a href=\"/%s\">%s/</a></li>", link, gf.Name)
		} else {
			link = fmt.Sprintf("%s/tree/%s/%s?branch=%s", repository, filePath, gf.Name, branchShort)
			text = fmt.Sprintf("<li><a href=\"/%s\">%s</a></li>", link, gf.Name)
		}
		ret += text
	}
	ret += "</ul></details>"

	return ret + "</html>"
}

func renderCommit(commit *object.Commit) (string, error) {
	parent, err := commit.Parents().Next()
	if err != nil {
		return "", err
	}

	parentTree, err := parent.Tree()
	if err != nil {
		return "", err
	}

	commitTree, err := commit.Tree()
	if err != nil {
		return "", nil
	}

	changes, err := object.DiffTreeWithOptions(context.Background(), parentTree, commitTree, &object.DiffTreeOptions{DetectRenames: true})
	if err != nil {
		return "", nil
	}

	patch, err := changes.Patch()
	if err != nil {
		return "", nil
	}

	return patch.String(), nil
}
