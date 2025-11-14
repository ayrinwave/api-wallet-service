import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Кастомные метрики
let conflictRate = new Rate('conflict_rate');
let errorRate = new Rate('error_rate');
let successRate = new Rate('success_rate');
let requestDuration = new Trend('request_duration');

export let options = {
    stages: [
        { duration: '10s', target: 100 },   // Разогрев: ~500 RPS
        { duration: '30s', target: 200 },   // Целевая нагрузка: ~1000 RPS
        { duration: '20s', target: 300 },   // Пик: ~1500 RPS
        { duration: '10s', target: 0 },     // Остывание
    ],
    thresholds: {
        'http_req_duration': ['p(95)<500', 'p(99)<1000'],
        'error_rate': ['rate<0.01'],  // Менее 1% ошибок (требование задания)
        'success_rate': ['rate>0.99'], // Более 99% успеха
    },
};

export default function () {
    const requestID = `k6-${__VU}-${__ITER}-${Date.now()}`;
    const url = 'http://localhost:8080/api/v1/wallet';

    const payload = JSON.stringify({
        walletId: '11111111-1111-1111-1111-111111111111',  // ✅ Один кошелёк
        operationType: 'DEPOSIT',
        amount: 1,
        requestID: requestID,
    });

    const params = {
        headers: { 'Content-Type': 'application/json' },
        timeout: '5s',
    };

    const startTime = Date.now();
    const res = http.post(url, payload, params);
    const duration = Date.now() - startTime;

    requestDuration.add(duration);

    const isSuccess = check(res, {
        'status is 200': (r) => r.status === 200,
        'no 5xx errors': (r) => r.status < 500,
        'response time OK': () => duration < 1000,
    });

    if (res.status === 200) {
        successRate.add(1);
    } else if (res.status >= 500) {
        errorRate.add(1);
        console.error(`ERROR [VU ${__VU}] Status ${res.status}: ${res.body}`);
    } else if (res.status === 409) {
        conflictRate.add(1);
    }

    // Задержка для контроля RPS
    sleep(0.1);  // 100ms = ~10 req/sec на VU
}

export function handleSummary(data) {
    console.log('\n=== SUMMARY ===');
    console.log(`Total Requests: ${data.metrics.http_reqs.values.count}`);
    console.log(`Success Rate: ${(data.metrics.success_rate.values.rate * 100).toFixed(2)}%`);
    console.log(`Error Rate: ${(data.metrics.error_rate.values.rate * 100).toFixed(2)}%`);
    console.log(`Avg Duration: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms`);
    console.log(`P95 Duration: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms`);
    console.log(`P99 Duration: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms`);

    return {
        'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    };
}