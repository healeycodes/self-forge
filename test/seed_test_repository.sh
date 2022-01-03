cd repositories
mkdir a_repository

cd a_repository
git init

git checkout -b one

echo "a" > a.txt 
git add .
git commit -m "Add a"

git checkout -b two

echo "b" > b.txt 
git add .
git commit -m "Add v"