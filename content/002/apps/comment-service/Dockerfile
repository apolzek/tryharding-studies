FROM openjdk:17-jdk-slim AS build

RUN apt-get update && apt-get install -y maven
WORKDIR /app
COPY pom.xml ./pom.xml
COPY src ./src
RUN mvn dependency:go-offline -B
RUN mvn clean package -DskipTests

FROM openjdk:17-jdk-slim
WORKDIR /app
COPY --from=build /app/target/*.jar app.jar
EXPOSE 8082
ENTRYPOINT ["java", "-jar", "app.jar"]