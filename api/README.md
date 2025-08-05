# Go Kafka Producer

Bu uygulama, REST API endpoint'i üzerinden gelen event verilerini Kafka topic'lerine yazan bir Go uygulamasıdır.

## Özellikler

- `/events` POST endpoint'i üzerinden JSON formatında event verilerini kabul eder
- Gelen verileri domain, subdomain ve code alanlarına göre otomatik topic isimlendirmesi yapar
- Geçersiz verileri filtreler ve uygun response döner
- Kafka'ya başarılı/başarısız yazma durumlarını takip eder

## API Kullanımı

### POST /events

Event verilerini Kafka'ya göndermek için kullanılır.

**Request Body:**
```json
[{
    "eventtimestamp": 1746788536758340000,
    "eventtime": "2025-05-09T14:02:16.75834+03:00",
    "id": "34B2D783-D297-D6B6-E063-4918060A0F70",
    "domain": "ForeignTrade",
    "subdomain": "Exchange",
    "code": "MoneyTransferOutgoingSwiftSent",
    "version": "1.0",
    "branchid": 8000,
    "channelid": 37,
    "customerid": 100537117,
    "userid": 78942,
    "payload": "..."
}]
```

**Response:**
```json
{
    "successEventIds": ["34B2D783-D297-D6B6-E063-4918060A0F70"],
    "invalidEventIds": [],
    "failedEventIds": []
}
```

### GET /protected/health

Uygulama sağlık durumunu kontrol etmek için kullanılır.

## Çevre Değişkenleri

- `PORT`: Uygulamanın çalışacağı port (varsayılan: 8080)
- `KAFKA_BROKERS`: Kafka broker adresleri, virgülle ayrılmış (varsayılan: localhost:9092)

## Çalıştırma

### Yerel Ortamda

```bash
# Bağımlılıkları yükle
go mod tidy

# Uygulamayı çalıştır
go run main.go
```

### Docker ile

```bash
# Docker image'ı build et
docker build -t go-kafka-producer .

# Container'ı çalıştır
docker run -p 8080:8080 \
  -e KAFKA_BROKERS=your-kafka-broker:9092 \
  go-kafka-producer
```

## Topic İsimlendirmesi

Topic isimleri otomatik olarak şu formatta oluşturulur:
```
{domain}_{subdomain}_{code}
```

Örnek: `Banking_Domestic_Created`

## Validasyon Kuralları

Bir event'in geçerli olması için aşağıdaki alanları dolu olmalıdır:
- `id`
- `domain`
- `subdomain`
- `code`

Bu alanlardan herhangi biri boş olan event'ler `invalidEventIds` listesine eklenir.

## Yük Testi

Uygulamanın performansını test etmek için entegre edilmiş bir yük testi aracı mevcuttur.

### Yük Testi Çalıştırma

```bash
# Basit test (30 saniye, 10 goroutine)
./loadtest.sh

# Özelleştirilmiş test
./loadtest.sh -d 60 -g 20 -e 5 -D 50

# Tüm seçenekleri görüntüle
./loadtest.sh --help
```

### Yük Testi Parametreleri

- `-d, --duration SECONDS`: Test süresi (varsayılan: 30)
- `-g, --goroutines NUMBER`: Eşzamanlı goroutine sayısı (varsayılan: 10)
- `-e, --events NUMBER`: İstek başına event sayısı (varsayılan: 1)
- `-D, --delay MILLISECONDS`: İstekler arası gecikme (varsayılan: 100)
- `-v, --verbose`: Detaylı çıktı
- `-u, --url URL`: API adresi (varsayılan: http://localhost:8080)

### Yük Testi Örnekleri

```bash
# 10 saniye hızlı test
./loadtest.sh -d 10 -g 5 -v

# 2 dakikalık yoğun test
./loadtest.sh -d 120 -g 50 -e 5 -D 50

# Yavaş ama uzun test
./loadtest.sh -d 300 -g 5 -e 1 -D 500
```

### Test Raporu

Yük testi şu metrikleri sağlar:
- **Request istatistikleri**: Toplam, başarılı, başarısız request sayıları
- **Event istatistikleri**: Başarılı/başarısız event sayıları
- **Latency istatistikleri**: Ortalama, minimum, maksimum yanıt süreleri
- **Throughput**: Saniye başına request ve event sayıları
- **Başarı oranları**: Request ve event bazında başarı yüzdeleri

## Test

### Fonksiyonel Test

```bash
# API testlerini çalıştır
./test.sh
```

### Yük Testi

```bash
# 10 saniyelik demo test
./loadtest.sh -d 10 -g 5 -e 2

# Production benzeri test
./loadtest.sh -d 300 -g 100 -e 10 -D 10
```
