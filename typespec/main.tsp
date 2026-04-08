import "@typespec/http";
import "@typespec/rest";
import "@typespec/openapi";
import "@typespec/openapi3";

import "./models.tsp";
import "./auth.tsp";
import "./users.tsp";
import "./schedules.tsp";
import "./groups.tsp";
import "./bookings.tsp";

using Http;
using Rest;

@service(#{ title: "Call Booking API" })
@server("http://localhost:8080", "Local development server")
namespace CallBooking;
