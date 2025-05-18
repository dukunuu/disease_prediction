import { type RouteConfig, index, route } from "@react-router/dev/routes";

export default [
  index("./routes/home.tsx"),
  route('/patients', "./routes/patients.tsx"),
  route("/patients/:patientId", './routes/patient.tsx')
] satisfies RouteConfig;
