FROM maven:3.9-eclipse-temurin-17 AS build
WORKDIR /src
COPY pom.xml ./
RUN mvn -q -B -DskipTests dependency:go-offline
COPY src ./src
RUN mvn -q -B -DskipTests package

FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
COPY --from=build /src/target/wh-server-jar-with-dependencies.jar app.jar
EXPOSE 9002
ENTRYPOINT ["java", "-jar", "/app/app.jar"]
