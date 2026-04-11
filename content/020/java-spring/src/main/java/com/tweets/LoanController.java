package com.tweets;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.*;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.util.UriComponentsBuilder;

import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/api/loan")
public class LoanController {

    private static final Logger log = LoggerFactory.getLogger(LoanController.class);

    private final CustomerRepository customerRepository;
    private final RestTemplate restTemplate;

    @Value("${python-fastapi.url}")
    private String pythonFastapiUrl;

    public LoanController(CustomerRepository customerRepository, RestTemplate restTemplate) {
        this.customerRepository = customerRepository;
        this.restTemplate = restTemplate;
    }

    @PostMapping("/enrich")
    public ResponseEntity<Map<String, Object>> enrich(
            @RequestBody Map<String, Object> request,
            @RequestHeader(value = "X-Application-ID", required = false) String applicationId,
            @RequestHeader(value = "X-Source-Service", required = false) String sourceService) {

        String customerId = (String) request.get("customer_id");
        log.info("Enriching loan application {} for customer {}", applicationId, customerId);

        // DB lookup — auto-instrumented by opentelemetry-spring-boot-starter (JDBC span)
        Customer customer = customerRepository.findByCustomerId(customerId).orElse(null);
        if (customer == null) {
            return ResponseEntity.status(HttpStatus.NOT_FOUND)
                    .body(Map.of("error", "Customer not found: " + customerId));
        }

        log.info("Customer {} found: credit_score={}, age_days={}",
                customerId, customer.getCreditScore(), customer.getAccountAgeDays());

        if (customer.getCreditScore() < 450) {
            return ResponseEntity.status(HttpStatus.UNPROCESSABLE_ENTITY)
                    .body(Map.of("error", "Credit score below minimum threshold (450)"));
        }

        // Build enriched payload and forward to python-fastapi
        // RestTemplate is auto-instrumented: injects W3C traceparent automatically
        Map<String, Object> enrichedPayload = new HashMap<>(request);
        enrichedPayload.put("credit_score", customer.getCreditScore());
        enrichedPayload.put("account_age_days", customer.getAccountAgeDays());
        enrichedPayload.put("monthly_income", customer.getMonthlyIncome());
        enrichedPayload.put("customer_name", customer.getName());

        String currency = (String) request.getOrDefault("currency", "USD");
        String targetUrl = UriComponentsBuilder
                .fromHttpUrl(pythonFastapiUrl + "/api/loan/compliance")
                .queryParam("currency", currency)
                .toUriString();

        HttpHeaders headers = new HttpHeaders();
        headers.setContentType(MediaType.APPLICATION_JSON);
        headers.set("X-Application-ID", applicationId != null ? applicationId : "");
        headers.set("X-Source-Service", "java-spring");

        log.info("Forwarding enriched application {} to python-fastapi (currency={})", applicationId, currency);

        ResponseEntity<Map> pythonResponse = restTemplate.exchange(
                targetUrl, HttpMethod.POST, new HttpEntity<>(enrichedPayload, headers), Map.class);

        Map<String, Object> finalResult = new HashMap<>();
        finalResult.put("application_id", applicationId);
        finalResult.put("customer_name", customer.getName());
        finalResult.put("credit_score", customer.getCreditScore());
        finalResult.put("compliance_and_scoring", pythonResponse.getBody());

        return ResponseEntity.ok(finalResult);
    }

    @GetMapping("/health")
    public ResponseEntity<Map<String, String>> health() {
        return ResponseEntity.ok(Map.of("status", "ok", "service", "java-spring"));
    }
}
