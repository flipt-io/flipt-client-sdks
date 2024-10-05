package io.flipt.client;

public class EvaluationException extends Exception {

  /** Constructs a new EvaluationException with {@code null} as its detail message. */
  public EvaluationException() {
    super();
  }

  /**
   * Constructs a new EvaluationException with the specified detail message.
   *
   * @param message the detail message.
   */
  public EvaluationException(String message) {
    super(message);
  }

  /**
   * Constructs a new EvaluationException with the specified detail message and cause.
   *
   * @param message the detail message.
   * @param cause the cause of the exception (a {@code null} value is permitted).
   */
  public EvaluationException(String message, Throwable cause) {
    super(message, cause);
  }

  /**
   * Constructs a new EvaluationException with the specified cause.
   *
   * @param cause the cause of the exception (a {@code null} value is permitted).
   */
  public EvaluationException(Throwable cause) {
    super(cause);
  }
}
