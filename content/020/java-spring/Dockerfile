FROM maven:3.9-eclipse-temurin-21 AS builder
WORKDIR /app

COPY pom.xml .
# Download OTel Java agent from Maven Central alongside app dependencies
RUN mvn dependency:go-offline -q && \
    mvn dependency:copy \
        -Dartifact=io.opentelemetry.javaagent:opentelemetry-javaagent:2.3.0 \
        -DoutputDirectory=/opt/agent -q

COPY src ./src
RUN mvn package -DskipTests -q

FROM eclipse-temurin:21-jre-jammy
WORKDIR /app

# The agent instruments Spring MVC, JDBC (Hibernate/HikariCP), RestTemplate
# and HTTP clients automatically — zero code changes needed in the app.
COPY --from=builder /opt/agent/opentelemetry-javaagent-2.3.0.jar opentelemetry-javaagent.jar
COPY --from=builder /app/target/*.jar app.jar

EXPOSE 8081
ENTRYPOINT ["java", "-javaagent:opentelemetry-javaagent.jar", "-jar", "app.jar"]
