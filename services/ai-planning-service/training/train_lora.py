from __future__ import annotations

from pathlib import Path

from training.artifacts import experiment_dir, sha256_path, write_experiment_manifest
from training.callbacks import JsonlMetricsCallback
from training.config import TrainingConfig
from training.dataset_loader import TrainingDatasetBundle
from training.formatter import format_sft_text


def run_lora_training(
    config: TrainingConfig,
    bundle: TrainingDatasetBundle,
    *,
    resume: bool = False,
) -> Path:
    try:
        from datasets import Dataset
        from peft import LoraConfig
        from transformers import AutoModelForCausalLM, AutoTokenizer, TrainingArguments
        from trl import SFTTrainer
    except ImportError as exc:
        raise RuntimeError(
            "Full local training requires services/ai-planning-service/requirements-training.txt"
        ) from exc

    output_dir = experiment_dir(config)
    train_dataset = Dataset.from_list(
        [{"text": format_sft_text(item)} for item in bundle.train_examples]
    )
    validation_dataset = Dataset.from_list(
        [{"text": format_sft_text(item)} for item in bundle.validation_examples]
    )
    tokenizer = AutoTokenizer.from_pretrained(
        config.base_model_name,
        revision=config.base_model_revision,
        use_fast=True,
    )
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
    model = AutoModelForCausalLM.from_pretrained(
        config.base_model_name,
        revision=config.base_model_revision,
        device_map="auto",
    )
    peft_config = LoraConfig(
        r=config.lora_r,
        lora_alpha=config.lora_alpha,
        lora_dropout=config.lora_dropout,
        bias="none",
        task_type="CAUSAL_LM",
        target_modules=config.target_modules,
    )
    args = TrainingArguments(
        output_dir=str(output_dir / "checkpoints"),
        per_device_train_batch_size=config.train_batch_size,
        per_device_eval_batch_size=config.eval_batch_size,
        gradient_accumulation_steps=config.gradient_accumulation_steps,
        num_train_epochs=config.num_train_epochs,
        learning_rate=config.learning_rate,
        warmup_ratio=config.warmup_ratio,
        weight_decay=config.weight_decay,
        eval_strategy="steps",
        eval_steps=50,
        save_steps=50,
        logging_steps=10,
        report_to=[],
        seed=config.seed,
        gradient_checkpointing=config.gradient_checkpointing,
    )
    trainer = SFTTrainer(
        model=model,
        tokenizer=tokenizer,
        train_dataset=train_dataset,
        eval_dataset=validation_dataset,
        peft_config=peft_config,
        dataset_text_field="text",
        max_seq_length=config.max_seq_length,
        args=args,
        callbacks=[JsonlMetricsCallback(output_dir / "metrics.jsonl")],
    )
    trainer.train(resume_from_checkpoint=resume)
    adapter_dir = output_dir / "adapter"
    trainer.model.save_pretrained(adapter_dir)
    tokenizer.save_pretrained(adapter_dir)
    checksum = sha256_path(adapter_dir)
    write_experiment_manifest(
        config,
        bundle,
        status="trained",
        extra={"adapterPath": str(adapter_dir), "adapterChecksum": checksum},
    )
    return adapter_dir
