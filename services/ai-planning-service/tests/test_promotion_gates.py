from training.metrics import evaluate_promotion_gates


def test_promotion_gates_approve_good_metrics() -> None:
    result = evaluate_promotion_gates(
        {
            "schemaValidRate": 1.0,
            "factualPrecision": 0.95,
            "groundingCitationRate": 0.98,
            "noPiiRate": 1.0,
            "costRegressionPct": 2.0,
            "latencyRegressionPct": 10.0,
            "holdoutUsedForTraining": False,
        }
    )

    assert result.approved is True
    assert result.failed_gates == []


def test_promotion_gates_reject_holdout_and_regressions() -> None:
    result = evaluate_promotion_gates(
        {
            "schemaValidRate": 0.95,
            "factualPrecision": 0.8,
            "groundingCitationRate": 0.9,
            "noPiiRate": 0.99,
            "costRegressionPct": 20.0,
            "latencyRegressionPct": 40.0,
            "holdoutUsedForTraining": True,
        }
    )

    assert result.approved is False
    assert "holdout_used_for_training" in result.failed_gates
    assert "schema_valid_rate" in result.failed_gates
