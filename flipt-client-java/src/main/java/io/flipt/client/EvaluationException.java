package io.flipt.client;

/** Base exception for all Flipt client errors. */
public class FliptException extends RuntimeException {
  public FliptException() {
    super();
  }

  public FliptException(String message) {
    super(message);
  }

  public FliptException(String message, Throwable cause) {
    super(message, cause);
  }

  public FliptException(Throwable cause) {
    super(cause);
  }

  /** Thrown when an evaluation error occurs. */
  public static final class EvaluationException extends FliptException {
    public EvaluationException() {
      super();
    }

    public EvaluationException(String message) {
      super(message);
    }

    public EvaluationException(String message, Throwable cause) {
      super(message, cause);
    }

    public EvaluationException(Throwable cause) {
      super(cause);
    }
  }

  /** Thrown when a configuration error occurs. */
  public static final class ConfigurationException extends FliptException {
    public ConfigurationException() {
      super();
    }

    public ConfigurationException(String message) {
      super(message);
    }

    public ConfigurationException(String message, Throwable cause) {
      super(message, cause);
    }

    public ConfigurationException(Throwable cause) {
      super(cause);
    }
  }
}
