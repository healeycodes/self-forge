cd repositories
mkdir a_repository

cd a_repository
git init

echo "a" > a.txt 
git add .
git commit -m "Add a"

echo "b" > b.txt 
git add .
git commit -m "Add b"

