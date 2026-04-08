using Http;
using Rest;

namespace CallBooking;

@route("/api/my/schedules")
interface Schedules {
  @get list(): SchedulesListResponse | ErrorResponse;

  @post create(@body schedule: CreateScheduleRequest): { @statusCode _: 201; ...Schedule } | ErrorResponse;

  @put
  @route("/{id}")
  update(@path id: string, @body schedule: CreateScheduleRequest): Schedule | ErrorResponse;

  @delete
  @route("/{id}")
  delete(@path id: string): { @statusCode _: 204; } | ErrorResponse;
}
