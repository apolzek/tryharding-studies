from locust import HttpUser, task, between


class BigBenchUser(HttpUser):
    wait_time = between(0, 0)

    @task(3)
    def simple(self):
        self.client.get("/simple", name="simple")

    @task(2)
    def medium(self):
        self.client.get("/medium?n=2000", name="medium")

    @task(1)
    def heavy(self):
        self.client.get("/heavy?n=25", name="heavy")
