# Load Test Application

Bu Go uygulaması Kafka Producer API'sini yük testi yapımak için geliştirilmiştir.

## Docker ile Çalıştırma

### 1. Sadece Ana Servisleri Çalıştırma (Kafka + API)
```bash
docker-compose up -d
```

### 2. Load Test ile Birlikte Çalıştırma
```bash
docker-compose --profile loadtest up
```

### 3. Sadece Load Test Çalıştırma (API zaten çalışıyorsa)
```bash
docker-compose run --rm loadtest
```

## Özelleştirme

docker-compose.yml dosyasındaki `loadtest` servisinin `command` bölümünü düzenleyerek parametreleri değiştirebilirsiniz:

```yaml
command: [
  "-duration", "60",        # Test süresi (saniye)
  "-goroutines", "5",       # Eşzamanlı goroutine sayısı
  "-url", "http://go-kafka-producer:8080",  # API URL
  "-events", "2",           # İstek başına event sayısı
  "-delay", "100",          # İstekler arası gecikme (ms)
  "-verbose"                # Detaylı çıktı
]
```

## Parametreler

- **duration**: Test süresi (saniye) - varsayılan: 30
- **goroutines**: Eşzamanlı çalışan goroutine sayısı - varsayılan: 10
- **url**: Test edilecek API'nin base URL'i - varsayılan: http://localhost:8080
- **events**: Her istekte gönderilecek event sayısı - varsayılan: 1
- **delay**: İstekler arası gecikme (milisaniye) - varsayılan: 100
- **verbose**: Detaylı çıktı için true/false - varsayılan: false

## Çıktı

Load test şu metrikleri sağlar:
- İstek istatistikleri (toplam, başarılı, başarısız)
- Event istatistikleri (toplam, başarılı, başarısız, geçersiz)
- Gecikme istatistikleri (ortalama, minimum, maksimum)
- Başarı oranları
- Saniye başına istek/event sayıları

## Örnek Kullanım

```bash
# Temel kullanım
docker-compose --profile loadtest up

# Özel parametrelerle
docker-compose run --rm loadtest -duration 120 -goroutines 10 -events 5 -verbose

# Local API'yi test etme
docker-compose run --rm loadtest -url http://host.docker.internal:8080 -duration 30
```
