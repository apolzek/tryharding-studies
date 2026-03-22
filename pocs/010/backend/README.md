curl http://localhost:3000/api/items
curl -X POST -H "Content-Type: application/json" -d '{"name":"Item1"}' http://localhost:3000/api/items
curl -X PUT -H "Content-Type: application/json" -d '{"name":"Updated Item"}' http://localhost:3000/api/items/0
curl -X DELETE http://localhost:3000/api/items/0
