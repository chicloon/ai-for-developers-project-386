using Http;
using Rest;

namespace CallBooking;

@route("/api/auth")
interface Auth {
  @post
  @route("/register")
  register(@body req: RegisterRequest): { @statusCode _: 201; ...AuthResponse } | ErrorResponse;

  @post
  @route("/login")
  login(@body req: LoginRequest): AuthResponse | ErrorResponse;

  @get
  @route("/me")
  me(): User | ErrorResponse;
}
