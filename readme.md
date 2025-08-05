--topic olusturma
docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --create --topic Banking_Domestic_Created --partitions 10 --replication-factor 1

--topic silme
docker-compose exec kafka kafka-topics --bootstrap-server localhost:9092 --delete --topic Banking_Domestic_Created

--topicteki mesaj sayisi

docker-compose exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic Banking_Domestic_Created \
  --time -1