import express from "express";
import router from "./routes/index.js";
import cors from "cors";
const app = express();
const port = process.env.PORT || 3000;

app.use(cors({ origin: "*" }));

app.get("/", (req, res) => {
  res.send("welcome to my multi stage kbernetes platform");
});
app.get("/api", (req, res) => {
  res.json({ message: "welcome to my multi stage kbernetes platform" });
});

app.use("/api", router);

app.listen(port, () => {
  console.log(`server is running on ${port}`);
});
