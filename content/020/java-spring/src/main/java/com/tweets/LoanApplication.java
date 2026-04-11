package com.tweets;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.context.annotation.Bean;
import org.springframework.web.client.RestTemplate;

@SpringBootApplication
public class LoanApplication {

    public static void main(String[] args) {
        SpringApplication.run(LoanApplication.class, args);
    }

    // RestTemplate is auto-instrumented by the OTel Spring Boot starter
    // to propagate trace context on outgoing HTTP calls
    @Bean
    public RestTemplate restTemplate() {
        return new RestTemplate();
    }
}
