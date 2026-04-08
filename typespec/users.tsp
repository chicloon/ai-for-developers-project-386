using Http;
using Rest;

namespace CallBooking;

@route("/api/users")
interface Users {
  @get list(): UsersListResponse | ErrorResponse;

  @get
  @route("/{id}")
  get(@path id: string): User | ErrorResponse;

  @get
  @route("/{id}/slots")
  slots(@path id: string, @query date: string): SlotsListResponse | ErrorResponse;
}
