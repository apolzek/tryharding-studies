package com.tweets;

import jakarta.persistence.*;
import java.math.BigDecimal;
import java.time.LocalDateTime;

@Entity
@Table(name = "customers")
public class Customer {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "customer_id", nullable = false, unique = true)
    private String customerId;

    @Column(nullable = false)
    private String name;

    @Column(name = "credit_score", nullable = false)
    private Integer creditScore;

    @Column(name = "account_age_days", nullable = false)
    private Integer accountAgeDays;

    @Column(name = "monthly_income", nullable = false)
    private BigDecimal monthlyIncome;

    @Column(name = "created_at")
    private LocalDateTime createdAt;

    // getters
    public Long getId()                    { return id; }
    public String getCustomerId()          { return customerId; }
    public String getName()               { return name; }
    public Integer getCreditScore()        { return creditScore; }
    public Integer getAccountAgeDays()     { return accountAgeDays; }
    public BigDecimal getMonthlyIncome()   { return monthlyIncome; }
    public LocalDateTime getCreatedAt()    { return createdAt; }
}
